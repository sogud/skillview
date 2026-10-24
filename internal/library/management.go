package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

type FileOperation struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Target   string `json:"target,omitempty"`
	Revision string `json:"revision"`
}

type ManagementPlan struct {
	Kind       string          `json:"kind"`
	Operations []FileOperation `json:"operations"`
	Conflicts  []string        `json:"conflicts"`
	Token      string          `json:"token"`
}

func (l *Library) writableRoot(path string) (*os.File, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", err
	}
	for _, source := range l.prefs.Sources {
		if !source.Approved || !source.Writable {
			continue
		}
		if _, inside := within(source.ResolvedPath, resolved); !inside {
			continue
		}
		root, err := openSource(source)
		if err != nil {
			return nil, "", err
		}
		root.Close()
		file, err := openAbsoluteDirectory(resolved)
		return file, resolved, err
	}
	return nil, "", fmt.Errorf("Allow editing for this folder first: %s", path)
}

func (l *Library) PlanManagement(home, kind string, platforms []string) (ManagementPlan, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.reloadPreferences(); err != nil {
		return ManagementPlan{}, err
	}
	return l.planManagement(home, kind, platforms)
}

func (l *Library) planManagement(home, kind string, platforms []string) (ManagementPlan, error) {
	plan := ManagementPlan{Kind: kind, Operations: []FileOperation{}, Conflicts: []string{}}
	if kind != "sync" && kind != "cleanup" {
		return plan, fmt.Errorf("Unknown management action.")
	}
	selected := []string{"agents", "claude"}
	if kind == "cleanup" && len(platforms) > 0 {
		selected = nil
	}
	for _, name := range platforms {
		if !hasString(selected, name) {
			selected = append(selected, name)
		}
	}
	shared, err := FindPlatform(home, "agents")
	if err != nil {
		return plan, err
	}
	claude, err := FindPlatform(home, "claude")
	if err != nil {
		return plan, err
	}
	for _, name := range selected {
		platform, err := FindPlatform(home, name)
		if err != nil {
			return plan, err
		}
		if platform.WritePath == "" {
			return plan, fmt.Errorf("This Agent has no writable Skill directory: %s", name)
		}
		root, resolved, err := l.writableRoot(platform.WritePath)
		if err != nil {
			return plan, err
		}
		parentID, err := descriptorIdentity(root)
		if err != nil {
			root.Close()
			return plan, err
		}
		if kind == "cleanup" {
			entries, err := root.ReadDir(-1)
			if err != nil {
				root.Close()
				return plan, err
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				stat, err := statAt(root, entry.Name())
				if err != nil {
					root.Close()
					return plan, err
				}
				path := filepath.Join(resolved, entry.Name())
				remove := false
				if stat.Mode&unix.S_IFMT == unix.S_IFLNK {
					_, err := os.Stat(path)
					remove = os.IsNotExist(err)
				} else if stat.Mode&unix.S_IFMT == unix.S_IFDIR {
					child, err := openChild(root, entry.Name(), true)
					if err != nil {
						root.Close()
						return plan, err
					}
					items, readErr := child.ReadDir(-1)
					child.Close()
					remove = readErr == nil && len(items) == 0
				}
				if remove {
					plan.Operations = append(plan.Operations, FileOperation{Kind: "trash", Path: path, Revision: fmt.Sprintf("%s:%d:%d:%d:%d", parentID, stat.Dev, stat.Ino, stat.Ctim.Sec, stat.Ctim.Nsec)})
				}
			}
		} else {
			origin := shared
			if name == "agents" {
				origin = claude
			}
			groups := map[string]map[string]bool{}
			for _, skill := range l.catalog.Skills {
				if !origin.Includes(skill) {
					continue
				}
				if groups[skill.Name] == nil {
					groups[skill.Name] = map[string]bool{}
				}
				if l.refs[skill.ID].fileName != "" {
					continue
				}
				groups[skill.Name][skill.Path] = true
			}
			for skillName, paths := range groups {
				if len(paths) != 1 || validateRelative(skillName) != nil || filepath.Base(skillName) != skillName || strings.HasPrefix(skillName, ".") {
					plan.Conflicts = append(plan.Conflicts, name+": "+skillName)
					continue
				}
				for sourcePath := range paths {
					path := filepath.Join(resolved, skillName)
					if actual, err := filepath.EvalSymlinks(path); err == nil && actual == sourcePath {
						continue
					}
					if _, err := statAt(root, skillName); !os.IsNotExist(err) {
						plan.Conflicts = append(plan.Conflicts, path)
						continue
					}
					sourceID, err := pathIdentity(sourcePath)
					if err != nil {
						root.Close()
						return plan, err
					}
					plan.Operations = append(plan.Operations, FileOperation{Kind: "link", Path: path, Target: sourcePath, Revision: parentID + ":" + sourceID})
				}
			}
		}
		root.Close()
	}
	sort.Slice(plan.Operations, func(i, j int) bool { return plan.Operations[i].Path < plan.Operations[j].Path })
	sort.Strings(plan.Conflicts)
	data, err := json.Marshal(plan)
	if err != nil {
		return plan, err
	}
	plan.Token = digest(data)
	return plan, nil
}

func (l *Library) ApplyManagement(home, kind string, platforms []string, expected string) (ManagementPlan, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return ManagementPlan{}, err
	}
	defer release()
	plan, err := l.planManagement(home, kind, platforms)
	if err != nil {
		return plan, err
	}
	if expected == "" || plan.Token != expected {
		return plan, fmt.Errorf("Files changed. Review this action again.")
	}
	for _, op := range plan.Operations {
		parent, err := openAbsoluteDirectory(filepath.Dir(op.Path))
		if err != nil {
			return plan, err
		}
		parentID, err := descriptorIdentity(parent)
		if err != nil || !strings.HasPrefix(op.Revision, parentID+":") {
			parent.Close()
			return plan, fmt.Errorf("Folder changed. Review this action again.")
		}
		if op.Kind == "link" {
			sourceID, sourceErr := pathIdentity(op.Target)
			if sourceErr != nil || op.Revision != parentID+":"+sourceID {
				parent.Close()
				return plan, fmt.Errorf("Skill changed. Review this action again.")
			}
			target, err := filepath.Rel(filepath.Dir(op.Path), op.Target)
			if err == nil {
				err = unix.Symlinkat(target, int(parent.Fd()), filepath.Base(op.Path))
			}
			parent.Close()
			if err != nil {
				return plan, err
			}
		} else {
			stat, err := statAt(parent, filepath.Base(op.Path))
			parent.Close()
			if err != nil {
				return plan, err
			}
			_, err = l.trashItem(TrashPreview{Path: op.Path, identity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)})
			if err != nil {
				return plan, err
			}
		}
	}
	return plan, nil
}
