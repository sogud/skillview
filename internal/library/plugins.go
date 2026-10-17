package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/semver"
	"golang.org/x/sys/unix"
)

type Plugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Agent       string `json:"agent"`
	Icon        string `json:"icon"`
	Enabled     *bool  `json:"enabled"`
}

type PluginCatalog struct {
	Plugins []Plugin `json:"plugins"`
	Issues  []string `json:"issues"`
}

type PluginHomes struct {
	Claude string
	Codex  string
}

func DefaultPluginHomes(home string) PluginHomes {
	return PluginHomes{Claude: filepath.Dir(expandLocation("$CLAUDE/skills", home)), Codex: filepath.Dir(expandLocation("$CODEX/skills", home))}
}

func pluginMetadata(path string, value any, tomlFile bool) error {
	file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 4*1024*1024 {
		return fmt.Errorf("plugin metadata is not a bounded ordinary file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 4*1024*1024 {
		return fmt.Errorf("plugin metadata is too large")
	}
	if tomlFile {
		return toml.Unmarshal(data, value)
	}
	return json.Unmarshal(data, value)
}

// Installation records, not the presence of old cache directories, select plugins.
func (l *Library) DiscoverPlugins(homes PluginHomes) PluginCatalog {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := PluginCatalog{Plugins: []Plugin{}, Issues: []string{}}
	l.plugins = map[string]Source{}
	add := func(agent, name, version, path string, enabled *bool) {
		source, ok := automaticSource(path, name, "plugin", []string{agent})
		if !ok {
			result.Issues = append(result.Issues, agent+": an installed plugin folder is unavailable.")
			return
		}
		id := objectID(agent + ":" + source.ResolvedPath)
		if _, exists := l.plugins[id]; exists {
			return
		}
		type pluginManifest struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Interface   struct {
				DisplayName string `json:"displayName"`
			} `json:"interface"`
		}
		var manifest pluginManifest
		for _, relative := range []string{".codex-plugin/plugin.json", ".claude-plugin/plugin.json"} {
			root, err := openAbsoluteDirectory(source.ResolvedPath)
			if err != nil {
				return
			}
			file, parent, _, err := openResource(root, relative)
			root.Close()
			if err != nil {
				continue
			}
			data, readErr := readRegularText(file)
			file.Close()
			parent.Close()
			if readErr == nil && json.Unmarshal(data, &manifest) == nil {
				break
			}
			manifest = pluginManifest{}
		}
		if manifest.DisplayName != "" {
			name = manifest.DisplayName
		} else if manifest.Interface.DisplayName != "" {
			name = manifest.Interface.DisplayName
		} else if manifest.Name != "" {
			name = manifest.Name
		}
		if version == "" {
			version = manifest.Version
		}
		if version == "" {
			version = filepath.Base(source.ResolvedPath)
		}
		l.plugins[id] = source
		result.Plugins = append(result.Plugins, Plugin{ID: id, Name: name, Version: version, Description: manifest.Description, Agent: agent, Icon: agentIcon(agent), Enabled: enabled})
	}
	if homes.Claude != "" {
		var installed struct {
			Plugins map[string][]struct {
				InstallPath string `json:"installPath"`
				Version     string `json:"version"`
			} `json:"plugins"`
		}
		err := pluginMetadata(filepath.Join(homes.Claude, "plugins/installed_plugins.json"), &installed, false)
		if err != nil {
			installed.Plugins = nil
			if !errors.Is(err, os.ErrNotExist) {
				result.Issues = append(result.Issues, "Claude Code: plugin installation records could not be read.")
			}
		}
		for id, entries := range installed.Plugins {
			name, _, _ := strings.Cut(id, "@")
			for _, entry := range entries {
				if entry.InstallPath != "" {
					add("Claude Code", name, entry.Version, entry.InstallPath, nil)
				}
			}
		}
	}
	if homes.Codex != "" {
		var config struct {
			Plugins map[string]struct {
				Enabled *bool  `toml:"enabled"`
				Version string `toml:"version"`
			} `toml:"plugins"`
		}
		err := pluginMetadata(filepath.Join(homes.Codex, "config.toml"), &config, true)
		if err != nil {
			config.Plugins = nil
			if !errors.Is(err, os.ErrNotExist) {
				result.Issues = append(result.Issues, "Codex: plugin installation records could not be read.")
			}
		}
		cache := filepath.Join(homes.Codex, "plugins/cache")
		for id, entry := range config.Plugins {
			name, marketplace, ok := strings.Cut(id, "@")
			if !ok || !pluginComponent(name) || !pluginComponent(marketplace) {
				continue
			}
			base := filepath.Join(cache, marketplace, name)
			path := pluginVersionRoot(base, entry.Version)
			if path != "" {
				add("Codex", name, entry.Version, path, entry.Enabled)
			} else {
				result.Issues = append(result.Issues, "Codex: an installed plugin folder is unavailable.")
			}
		}
		// Desktop remote installs carry an installation marker even when they are
		// absent from CLI configuration. Never treat an unmarked cache as installed.
		markets, readErr := os.ReadDir(cache)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			result.Issues = append(result.Issues, "Codex: installed plugin folders could not be read.")
		}
		for _, market := range markets {
			if !market.IsDir() {
				continue
			}
			entries, err := os.ReadDir(filepath.Join(cache, market.Name()))
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if _, exists := config.Plugins[entry.Name()+"@"+market.Name()]; exists {
					continue
				}
				base := filepath.Join(cache, market.Name(), entry.Name())
				var marker struct {
					Schema int    `json:"schema_version"`
					ID     string `json:"remote_plugin_id"`
				}
				if pluginMetadata(filepath.Join(base, ".codex-remote-plugin-install.json"), &marker, false) != nil || marker.Schema != 1 || marker.ID == "" {
					continue
				}
				if path := pluginVersionRoot(base, ""); path != "" {
					add("Codex", entry.Name(), "", path, nil)
				}
			}
		}
	}
	sort.Slice(result.Plugins, func(i, j int) bool {
		if result.Plugins[i].Name != result.Plugins[j].Name {
			return result.Plugins[i].Name < result.Plugins[j].Name
		}
		return result.Plugins[i].ID < result.Plugins[j].ID
	})
	return result
}

func pluginComponent(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

func pluginVersionRoot(base, version string) string {
	isPluginRoot := func(path string) bool {
		for _, manifest := range []string{".codex-plugin/plugin.json", ".claude-plugin/plugin.json"} {
			info, err := os.Lstat(filepath.Join(path, manifest))
			if err == nil && info.Mode().IsRegular() {
				return true
			}
		}
		return false
	}
	if version != "" {
		if !pluginComponent(version) {
			return ""
		}
		path := filepath.Join(base, version)
		if info, err := os.Stat(path); err == nil && info.IsDir() && isPluginRoot(path) {
			return path
		}
		return ""
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return ""
	}
	var chosen string
	var latest int64
	for _, entry := range entries {
		if !entry.IsDir() || !isPluginRoot(filepath.Join(base, entry.Name())) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidate := "v" + strings.TrimPrefix(entry.Name(), "v")
		current := "v" + strings.TrimPrefix(chosen, "v")
		if chosen == "" || (semver.IsValid(candidate) && semver.IsValid(current) && semver.Compare(candidate, current) > 0) || ((!semver.IsValid(candidate) || !semver.IsValid(current)) && info.ModTime().UnixNano() > latest) {
			chosen = entry.Name()
			latest = info.ModTime().UnixNano()
		}
	}
	if chosen == "" {
		return ""
	}
	return filepath.Join(base, chosen)
}

func (l *Library) openPlugin(id string) (*os.File, error) {
	source, exists := l.plugins[id]
	if !exists {
		return nil, fmt.Errorf("Plugin is no longer installed. Refresh the plugin list.")
	}
	resolved, err := filepath.EvalSymlinks(source.Path)
	if err != nil || resolved != source.ResolvedPath {
		return nil, fmt.Errorf("Plugin folder changed. Refresh the plugin list.")
	}
	root, err := openAbsoluteDirectory(resolved)
	if err != nil {
		return nil, fmt.Errorf("Plugin folder is unavailable. Refresh the plugin list.")
	}
	identity, err := descriptorIdentity(root)
	if err != nil || identity != source.Identity {
		root.Close()
		return nil, fmt.Errorf("Plugin folder changed. Refresh the plugin list.")
	}
	return root, nil
}

func (l *Library) PluginResources(id string) ([]Resource, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	root, err := l.openPlugin(id)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return directoryResources(root, true)
}

func (l *Library) ReadPlugin(id, path string) (Document, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	root, err := l.openPlugin(id)
	if err != nil {
		return Document{}, err
	}
	defer root.Close()
	file, parent, _, err := openResource(root, path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()
	defer parent.Close()
	doc, err := document(file, id, path, false)
	doc.ReadOnlyReason = "Installed plugin files are read-only."
	return doc, err
}
