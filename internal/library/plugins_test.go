package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func pluginFixture(t *testing.T, root, name, version string) string {
	t.Helper()
	path := filepath.Join(root, name, version)
	for _, folder := range []string{".codex-plugin", "skills/example", "commands", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(path, folder), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for relative, content := range map[string]string{
		".codex-plugin/plugin.json": `{"name":"` + name + `","version":"` + version + `","description":"Fixture plugin"}`,
		"README.md":                 "# Plugin guide\n\nRead only.\n",
		"skills/example/SKILL.md":   fixture,
		"commands/review.md":        "# Review command\n",
		"node_modules/ignored.js":   "throw new Error('must not run')",
	} {
		if err := os.WriteFile(filepath.Join(path, relative), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestPluginsSelectInstallationRecordsAndRemoteMarkers(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, "claude")
	codex := filepath.Join(home, "codex")
	claudePath := pluginFixture(t, filepath.Join(claude, "plugins/cache/market"), "claude-example", "2.0.0")
	pluginFixture(t, filepath.Join(claude, "plugins/cache/market"), "old-cache", "1.0.0")
	data, _ := json.Marshal(map[string]any{"version": 2, "plugins": map[string]any{"claude-example@market": []any{map[string]string{"installPath": claudePath, "version": "2.0.0"}}}})
	if err := os.WriteFile(filepath.Join(claude, "plugins/installed_plugins.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(codex, "plugins/cache/market")
	pluginFixture(t, base, "codex-example", "1.0.2")
	pluginFixture(t, base, "codex-example", "1.0.10")
	if err := os.Mkdir(filepath.Join(base, "codex-example", "staging"), 0755); err != nil {
		t.Fatal(err)
	}
	pluginFixture(t, base, "not-installed", "9.0.0")
	remote := pluginFixture(t, base, "desktop-example", "3.0.0")
	if err := os.WriteFile(filepath.Join(filepath.Dir(remote), ".codex-remote-plugin-install.json"), []byte(`{"schema_version":1,"remote_plugin_id":"fixture-remote"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("[plugins.\"codex-example@market\"]\nenabled = false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	l, _ := New(t.TempDir(), nil)
	c := l.DiscoverPlugins(PluginHomes{Claude: claude, Codex: codex})
	if len(c.Plugins) != 3 || len(c.Issues) != 0 {
		t.Fatalf("plugins=%+v issues=%v", c.Plugins, c.Issues)
	}
	for _, p := range c.Plugins {
		if p.Name == "codex-example" && (p.Version != "1.0.10" || p.Enabled == nil || *p.Enabled) {
			t.Fatal("disabled installation/version lost", p)
		}
		files, err := l.PluginResources(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		foundManifest := false
		for _, file := range files {
			if file.Path == ".codex-plugin/plugin.json" {
				foundManifest = true
			}
			if file.Path == "node_modules/ignored.js" {
				t.Fatal("dependency tree included")
			}
		}
		if !foundManifest {
			t.Fatal("manifest missing")
		}
		doc, err := l.ReadPlugin(p.ID, "README.md")
		if err != nil || doc.Editable {
			t.Fatal("plugin not read-only", err)
		}
		if _, err = l.Save(p.ID, "README.md", doc.Revision, "replace"); err == nil {
			t.Fatal("plugin ID accepted by Skill editor")
		}
		if _, err = l.ReadPlugin(p.ID, "../config.toml"); err == nil {
			t.Fatal("parent traversal allowed")
		}
	}
}

func TestPluginsRejectLinksReplacedRootsAndRemovedInstallations(t *testing.T) {
	home := t.TempDir()
	root := pluginFixture(t, filepath.Join(home, "plugins/cache/market"), "sample", "1.0.0")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[plugins.\"sample@market\"]\nenabled=true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	l, _ := New(t.TempDir(), nil)
	c := l.DiscoverPlugins(PluginHomes{Codex: home})
	id := c.Plugins[0].ID
	outside := filepath.Join(t.TempDir(), "private.md")
	_ = os.WriteFile(outside, []byte("outside"), 0644)
	_ = os.Symlink(outside, filepath.Join(root, "link.md"))
	if _, err := l.ReadPlugin(id, "link.md"); err == nil {
		t.Fatal("resource link followed")
	}
	if err := os.Rename(root, root+"-old"); err != nil {
		t.Fatal(err)
	}
	_ = os.Mkdir(root, 0755)
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("replaced"), 0644)
	if _, err := l.ReadPlugin(id, "README.md"); err == nil {
		t.Fatal("replaced root read")
	}
	_ = os.WriteFile(filepath.Join(home, "config.toml"), []byte(""), 0644)
	c = l.DiscoverPlugins(PluginHomes{Codex: home})
	if len(c.Plugins) != 0 {
		t.Fatal("cache-only plugin shown")
	}
	if _, err := l.ReadPlugin(id, "README.md"); err == nil {
		t.Fatal("removed ID still authorized")
	}
}

func TestPluginRegistryFailuresAreVisibleAndDoNotLeakContents(t *testing.T) {
	home := t.TempDir()
	_ = os.WriteFile(filepath.Join(home, "config.toml"), []byte("[plugins.\"bad@market\"]\nenabled=fixture-private-value"), 0644)
	l, _ := New(t.TempDir(), nil)
	c := l.DiscoverPlugins(PluginHomes{Codex: home})
	if len(c.Plugins) != 0 || len(c.Issues) != 1 || c.Issues[0] != "Codex: plugin installation records could not be read." {
		t.Fatal(c)
	}
	if len(l.DiscoverPlugins(PluginHomes{}).Plugins) != 0 {
		t.Fatal("isolated empty homes read real plugins")
	}
}

func TestPluginPinnedVersionAndInvalidManifest(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, "plugins/cache/market")
	root := pluginFixture(t, base, "sample", "1.0.0")
	pluginFixture(t, base, "sample", "2.0.0")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[plugins.\"sample@market\"]\nenabled=true\nversion=\"1.0.0\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Invalid metadata must not leak partially decoded display names.
	if err := os.WriteFile(filepath.Join(root, ".codex-plugin/plugin.json"), []byte(`{"name":"wrong", "version":42}`), 0644); err != nil {
		t.Fatal(err)
	}
	l, _ := New(t.TempDir(), nil)
	c := l.DiscoverPlugins(PluginHomes{Codex: home})
	if len(c.Plugins) != 1 || c.Plugins[0].Version != "1.0.0" || c.Plugins[0].Name != "sample" {
		t.Fatalf("pinned version or metadata fallback failed: %+v", c)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	c = l.DiscoverPlugins(PluginHomes{Codex: home})
	if len(c.Plugins) != 0 || len(c.Issues) != 1 {
		t.Fatalf("missing pinned version fell back silently: %+v", c)
	}
}
