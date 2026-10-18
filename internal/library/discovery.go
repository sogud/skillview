package library

import (
	_ "embed"
	"encoding/json"
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
			if strings.Contains(path, "/plugins/") || strings.Contains(path, "/extensions") || strings.Contains(path, "/bundled-marketplace") {
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
			result[i].Name = "Common Skills"
			result[i].Tools = append(result[i].Tools, "Shared")
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func (l *Library) mergeAutomatic(candidates []Source) {
	positions := map[string]int{}
	known := map[string]bool{}
	for _, candidate := range candidates {
		known[candidate.ID] = true
	}
	kept := []Source{}
	for _, source := range l.prefs.Sources {
		// Old versions stored non-existent automatic candidates without approval.
		if (!source.Approved && !source.Writable) || (source.Automatic && !source.Writable && !known[source.ID]) {
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
			if !previous.Automatic && previous.Approved && previous.Identity == candidate.Identity {
				candidate.Writable = previous.Writable
				candidate.Automatic = false
			}
			l.prefs.Sources[index] = candidate
		} else {
			positions[candidate.ID] = len(l.prefs.Sources)
			l.prefs.Sources = append(l.prefs.Sources, candidate)
		}
	}
}

// RefreshSources revisits only registry locations; it never walks Home or volumes.
func (l *Library) RefreshSources(candidates []Source) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	previous := l.prefs.Sources
	l.mergeAutomatic(candidates)
	if err := l.persist(); err != nil {
		l.prefs.Sources = previous
		return err
	}
	return nil
}

func skipDiscoveryDirectory(name string) bool {
	if name == ".skillview-archive" || strings.HasPrefix(name, ".skillview-archiving-") {
		return true
	}
	switch name {
	case ".git", "node_modules", ".artifacts", ".Trash", ".Trashes", ".Spotlight-V100", ".fseventsd", ".npm", ".cache", ".venv", "venv", "__pycache__", ".build", "DerivedData":
		return true
	}
	return strings.HasSuffix(name, ".app") || strings.HasSuffix(name, ".photoslibrary") || strings.HasSuffix(name, ".backupbundle") || strings.HasSuffix(name, ".sparsebundle")
}
