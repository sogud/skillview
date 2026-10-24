package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/sys/unix"
)

type InspectionEntry struct {
	Path   string `json:"path"`
	Target string `json:"target,omitempty"`
	Broken bool   `json:"broken,omitempty"`
	Issue  string `json:"issue,omitempty"`
}

type Inspection struct {
	Entries    []InspectionEntry   `json:"entries"`
	Issues     []Issue             `json:"issues"`
	Duplicates map[string][]string `json:"duplicates"`
	NeedsScan  bool                `json:"needsScan"`
}

func (l *Library) Inspect(health bool) (Inspection, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := Inspection{Entries: []InspectionEntry{}, Issues: l.catalog.Issues, Duplicates: map[string][]string{}, NeedsScan: l.catalog.NeedsScan}
	if err := l.reloadPreferences(); err != nil {
		return result, err
	}
	for _, source := range l.prefs.Sources {
		root, err := openSource(source)
		if err != nil {
			result.Entries = append(result.Entries, InspectionEntry{Path: source.Path, Issue: "Folder unavailable or changed"})
			continue
		}
		entries, err := root.ReadDir(-1)
		if err != nil {
			root.Close()
			return result, err
		}
		for _, entry := range entries {
			if skipDiscoveryDirectory(entry.Name()) {
				continue
			}
			path := filepath.Join(source.ResolvedPath, entry.Name())
			stat, err := statAt(root, entry.Name())
			if err != nil {
				root.Close()
				return result, err
			}
			if stat.Mode&unix.S_IFMT == unix.S_IFLNK {
				buffer := make([]byte, 4096)
				n, err := unix.Readlinkat(int(root.Fd()), entry.Name(), buffer)
				if err != nil || n == len(buffer) {
					root.Close()
					return result, fmt.Errorf("Could not inspect link.")
				}
				_, targetErr := os.Stat(path)
				result.Entries = append(result.Entries, InspectionEntry{Path: path, Target: string(buffer[:n]), Broken: os.IsNotExist(targetErr)})
			} else if health && entry.IsDir() {
				child, err := openChild(root, entry.Name(), true)
				if err != nil {
					root.Close()
					return result, err
				}
				items, err := child.ReadDir(-1)
				child.Close()
				if err != nil {
					root.Close()
					return result, err
				}
				if len(items) == 0 {
					result.Entries = append(result.Entries, InspectionEntry{Path: path, Issue: "Empty folder"})
					continue
				}
				containsSkill := false
				for _, skill := range l.catalog.Skills {
					if _, inside := within(path, skill.Path); inside {
						containsSkill = true
						break
					}
				}
				if !containsSkill {
					result.Entries = append(result.Entries, InspectionEntry{Path: path, Issue: "No catalogued Skill; scan or inspect this folder"})
				}
			}
		}
		root.Close()
	}
	if health {
		byName := map[string][]string{}
		for _, skill := range l.catalog.Skills {
			byName[skill.Name] = append(byName[skill.Name], skill.Path)
		}
		for name, paths := range byName {
			if len(paths) > 1 {
				sort.Strings(paths)
				result.Duplicates[name] = paths
			}
		}
	}
	return result, nil
}
