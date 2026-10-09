package library

import (
	"context"
	_ "embed"
	"encoding/json"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Directory facts from upstream registries and vendor documentation. See docs/sources.md.
//
//go:embed locations.json
var locationData []byte

type AgentLocation struct {
	Tool    string   `json:"tool"`
	Global  []string `json:"global"`
	Project []string `json:"project"`
	Detect  []string `json:"detect"`
	Apps    []string `json:"apps"`
	Bins    []string `json:"bins"`
}

func Locations() []AgentLocation {
	var locations []AgentLocation
	if err := json.Unmarshal(locationData, &locations); err != nil {
		panic(err)
	}
	return locations
}

func expandLocation(path, home string) string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	values := map[string]string{"~": home, "$CONFIG": config}
	for token, variable := range map[string]string{"$CODEX": "CODEX_HOME", "$CLAUDE": "CLAUDE_CONFIG_DIR", "$HERMES": "HERMES_HOME", "$VIBE": "VIBE_HOME", "$AUTOHAND": "AUTOHAND_HOME", "$GROK": "GROK_HOME", "$SARVAM": "SARVAM_HOME"} {
		value := os.Getenv(variable)
		if value == "" {
			value = filepath.Join(home, "."+strings.ToLower(strings.TrimPrefix(token, "$")))
		}
		values[token] = value
	}
	for token, value := range values {
		if strings.HasPrefix(path, token+"/") {
			return filepath.Join(value, strings.TrimPrefix(path, token+"/"))
		}
	}
	return filepath.Clean(path)
}

func automaticSource(path, name, scope string, tools []string) (Source, bool) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Source{}, false
	}
	identity, err := pathIdentity(resolved)
	if err != nil {
		return Source{}, false
	}
	return Source{ID: objectID(path), Name: name, Path: path, ResolvedPath: resolved, Identity: identity, Tools: tools, Approved: true, Automatic: true, Scope: scope}, true
}

// Only directories that exist are shown. Shared directories do not imply that
// every compatible client is installed.
func DefaultSources(home string) []Source {
	result := []Source{}
	positions := map[string]int{}
	for _, location := range Locations() {
		for _, pattern := range location.Global {
			path := expandLocation(pattern, home)
			if index, exists := positions[path]; exists {
				if !hasString(result[index].Tools, location.Tool) {
					result[index].Tools = append(result[index].Tools, location.Tool)
				}
				continue
			}
			scope := "global"
			if strings.Contains(path, "/plugins/") || strings.Contains(path, "/extensions") {
				scope = "plugin"
			}
			source, ok := automaticSource(path, location.Tool, scope, []string{location.Tool})
			if !ok {
				continue
			}
			positions[path] = len(result)
			result = append(result, source)
		}
	}
	for i := range result {
		if len(result[i].Tools) > 2 || filepath.Base(filepath.Dir(result[i].Path)) == ".agents" || strings.HasSuffix(result[i].Path, "/.config/agents/skills") {
			result[i].Name = "Shared Skills"
			result[i].Tools = append(result[i].Tools, "Shared")
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func (l *Library) mergeAutomatic(candidates []Source) {
	positions := map[string]int{}
	kept := []Source{}
	for _, source := range l.prefs.Sources {
		// Old versions stored non-existent automatic candidates without approval.
		if !source.Approved && !source.Writable {
			continue
		}
		positions[source.ID] = len(kept)
		kept = append(kept, source)
	}
	l.prefs.Sources = kept
	for _, candidate := range candidates {
		if l.prefs.ExcludedSources[candidate.ID] {
			continue
		}
		if index, exists := positions[candidate.ID]; exists {
			previous := l.prefs.Sources[index]
			if previous.Writable && previous.Identity == candidate.Identity {
				candidate.Writable = true
				candidate.Automatic = false
			}
			l.prefs.Sources[index] = candidate
		} else {
			positions[candidate.ID] = len(l.prefs.Sources)
			l.prefs.Sources = append(l.prefs.Sources, candidate)
		}
	}
}

// File-name discovery reads directory metadata only. Skill contents are read by
// the normal bounded catalog scanner after a source has been discovered.
func (l *Library) Discover(ctx context.Context, roots []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return l.discover(ctx, roots, home, DefaultSources(home))
}

func (l *Library) discover(ctx context.Context, roots []string, home string, candidates []Source) error {
	known := map[string]bool{}
	for _, source := range candidates {
		known[source.ResolvedPath] = true
	}
	projectPaths := map[string][]string{}
	installed := map[string]bool{}
	for _, agent := range Agents(home) {
		installed[agent.Name] = agent.Installed
	}
	for _, location := range Locations() {
		for _, path := range location.Project {
			if _, exists := projectPaths[path]; !exists {
				projectPaths[path] = []string{}
			}
			if path == ".agents/skills" && !installed[location.Tool] {
				continue
			}
			projectPaths[path] = append(projectPaths[path], location.Tool)
		}
	}
	report := DiscoveryReport{Roots: append([]string{}, roots...), Completed: true}
	seen := map[string]bool{}
	var walk func(string, int)
	walk = func(path string, depth int) {
		if ctx.Err() != nil || report.Limited {
			return
		}
		if depth > 64 || report.Directories >= 250000 {
			report.Limited = true
			report.Completed = false
			return
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return
		}
		if seen[resolved] {
			return
		}
		seen[resolved] = true
		report.Directories++
		if known[resolved] {
			return
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			if os.IsPermission(err) {
				report.Denied++
			}
			return
		}
		entryNames := map[string]bool{}
		for _, entry := range entries {
			entryNames[entry.Name()] = true
			if entry.Name() == "SKILL.md" && entry.Type().IsRegular() {
				source, ok := automaticSource(path, filepath.Base(path), "local", []string{"Local"})
				if ok {
					candidates = append(candidates, source)
					known[resolved] = true
				}
				return
			}
		}
		for relative, tools := range projectPaths {
			first := strings.SplitN(relative, "/", 2)[0]
			if !entryNames[first] {
				continue
			}
			candidatePath := filepath.Join(resolved, relative)
			if source, ok := automaticSource(candidatePath, filepath.Base(resolved)+" · "+relative, "project", tools); ok {
				if relative == ".agents/skills" || len(tools) > 2 {
					source.Tools = append(source.Tools, "Shared")
				}
				if !known[source.ResolvedPath] {
					candidates = append(candidates, source)
					known[source.ResolvedPath] = true
				}
			}
		}
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() || skipDiscoveryDirectory(name) {
				continue
			}
			// Home's Library is scanned only through known Agent locations above.
			if resolved == home && name == "Library" {
				continue
			}
			walk(filepath.Join(resolved, name), depth+1)
		}
	}
	for _, root := range roots {
		walk(root, 0)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mergeAutomatic(candidates)
	l.discovery = report
	return l.persist()
}

func skipDiscoveryDirectory(name string) bool {
	switch name {
	case ".git", "node_modules", ".artifacts", ".Trash", ".Trashes", ".Spotlight-V100", ".fseventsd", ".npm", ".cache", ".venv", "venv", "__pycache__", ".build", "DerivedData":
		return true
	}
	return strings.HasSuffix(name, ".app") || strings.HasSuffix(name, ".photoslibrary") || strings.HasSuffix(name, ".backupbundle") || strings.HasSuffix(name, ".sparsebundle")
}

func DefaultSearchRoots(home string) []string {
	roots := []string{home}
	// External development volumes are searched where people commonly keep code.
	// Root-level system/backup folders are not traversed.
	volumes, _ := os.ReadDir("/Volumes")
	var system unix.Statfs_t
	_ = unix.Statfs("/", &system)
	localVolumes := []string{}
	for _, volume := range volumes {
		volumePath := filepath.Join("/Volumes", volume.Name())
		var stat unix.Statfs_t
		if unix.Statfs(volumePath, &stat) != nil || stat.Flags&unix.MNT_LOCAL == 0 || stat.Fsid == system.Fsid {
			continue
		}
		localVolumes = append(localVolumes, volumePath)
		for _, name := range []string{"workspace", "projects", "Projects", "code", "Developer", "dev", "src"} {
			path := filepath.Join("/Volumes", volume.Name(), name)
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				roots = append(roots, path)
			}
		}
	}
	return append(roots, localVolumes...)
}
