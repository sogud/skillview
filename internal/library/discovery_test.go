package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKnownSourcesExcludeArbitraryProjectsAndRetireOldAutomaticSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	for _, path := range []string{".codex/skills/example", ".codex/.tmp/bundled-marketplace/plugins/sample/skills/bundled", "projects/arbitrary/lesson"} {
		dir := filepath.Join(home, path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0644); err != nil {
			t.Fatal(err)
		}
	}
	data := t.TempDir()
	l, _ := New(data, nil)
	old, _ := automaticSource(filepath.Join(home, "projects"), "Old discovered projects", "local", []string{"Local"})
	l.prefs.Sources = []Source{old}
	if err := l.persist(); err != nil {
		t.Fatal(err)
	}
	l, _ = New(data, DefaultSources(home))
	catalog, err := l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 2 || len(catalog.Sources) != 2 {
		t.Fatalf("known scan: skills=%d folders=%d %v", len(catalog.Skills), len(catalog.Sources), err)
	}
	for _, skill := range catalog.Skills {
		if strings.Contains(skill.Path, "/projects/") || !hasString(skill.Tools, "Codex") {
			t.Fatal("unrelated source scanned", skill.Tools)
		}
	}
	if _, err := l.AddSource(filepath.Join(home, "projects"), false); err != nil {
		t.Fatal(err)
	}
	if err := l.RefreshSources(DefaultSources(home)); err != nil {
		t.Fatal(err)
	}
	catalog, err = l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 3 {
		t.Fatal("explicit directory was lost", err)
	}
}

func TestAutomaticLinksReadOnlyDeduplicatedAndRetargetRefused(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents/skills")
	_ = os.MkdirAll(root, 0755)
	actual := t.TempDir()
	dir := filepath.Join(actual, "sample")
	_ = os.Mkdir(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0644)
	alias := filepath.Join(root, "alias")
	_ = os.Symlink(dir, alias)
	source, ok := automaticSource(root, "Shared Skills", "global", []string{"Shared"})
	if !ok {
		t.Fatal("source")
	}
	l, _ := New(t.TempDir(), []Source{source})
	c, err := l.Scan(context.Background())
	if err != nil || len(c.Skills) != 1 {
		t.Fatal("automatic link missing", err)
	}
	id := c.Skills[0].ID
	doc, err := l.Read(id, "SKILL.md")
	if err != nil || doc.Editable {
		t.Fatal("linked read/edit", err)
	}
	if _, err = l.Save(id, "SKILL.md", doc.Revision, "replace"); err == nil {
		t.Fatal("linked write allowed")
	}
	_, _ = l.AddSource(actual, true)
	c, err = l.Scan(context.Background())
	if err != nil || len(c.Skills) != 1 || len(c.Skills[0].Occurrences) != 2 || !c.Skills[0].Editable {
		t.Fatal("physical dedup/access", err)
	}
	// Verify a cached linked reference cannot read a newly retargeted directory.
	l2, _ := New(t.TempDir(), []Source{source})
	_, _ = l2.Scan(context.Background())
	other := t.TempDir()
	_ = os.WriteFile(filepath.Join(other, "SKILL.md"), []byte(fixture+"outside"), 0644)
	_ = os.Remove(alias)
	_ = os.Symlink(other, alias)
	if _, err = l2.Read(id, "SKILL.md"); err == nil {
		t.Fatal("retargeted link followed")
	}
}

func TestExcludedAutomaticSourceStaysExcludedAfterRestart(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".codex/skills")
	_ = os.MkdirAll(root, 0755)
	source, _ := automaticSource(root, "Codex", "global", []string{"Codex"})
	data := t.TempDir()
	l, _ := New(data, []Source{source})
	if err := l.RemoveSource(source.ID); err != nil {
		t.Fatal(err)
	}
	restored, _ := New(data, []Source{source})
	if len(restored.Settings().Sources) != 0 {
		t.Fatal("removed source restored automatically")
	}
	if _, err := restored.AddSource(root, true); err != nil {
		t.Fatal(err)
	}
	restored, _ = New(data, []Source{source})
	if len(restored.Settings().Sources) != 1 || !restored.Settings().Sources[0].Writable {
		t.Fatal("explicit permission lost")
	}
}

func TestSharedGlobalRootRetainsDeclaredAgentMembership(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".agents/skills")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, source := range DefaultSources(home) {
		if source.Path != root {
			continue
		}
		found = true
		if !hasString(source.Tools, "Shared") || !hasString(source.Tools, "Codex") || !hasString(source.Tools, "Pi") || !hasString(source.Tools, "fx") {
			t.Fatal("shared root lost declared consumers", source.Tools)
		}
	}
	if !found {
		t.Fatal("shared root missing")
	}
}

func TestFxSharedAndLegacySkillRoots(t *testing.T) {
	home := t.TempDir()
	for _, path := range []string{".agents/skills", ".fx/skills"} {
		if err := os.MkdirAll(filepath.Join(home, path), 0755); err != nil {
			t.Fatal(err)
		}
	}
	found := 0
	for _, source := range DefaultSources(home) {
		if hasString(source.Tools, "fx") {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("fx roots: got %d, want shared and legacy", found)
	}
	for _, location := range Locations() {
		if location.Tool == "fx" && (!hasString(location.Project, ".agents/skills") || !hasString(location.Project, ".fx/skills")) {
			t.Fatal("fx project roots missing", location.Project)
		}
	}
}
