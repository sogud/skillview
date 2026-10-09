package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryFindsProjectsCustomSkillsAndSkipsDependencies(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "projects")
	for _, path := range []string{"one/.cursor/skills/nested/sample", "two/.agents/skills/shared", "custom/deep/lesson", "node_modules/hidden"} {
		dir := filepath.Join(root, path)
		_ = os.MkdirAll(dir, 0755)
		_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0644)
	}
	l, _ := New(t.TempDir(), nil)
	if err := l.discover(context.Background(), []string{root}, home, nil); err != nil {
		t.Fatal(err)
	}
	c, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Skills) != 3 {
		t.Fatalf("discovered %d instead of 3", len(c.Skills))
	}
	shared := false
	cursor := false
	for _, s := range c.Skills {
		if s.Editable || strings.Contains(s.Path, "node_modules") {
			t.Fatal("automatic source writable or dependency scanned")
		}
		shared = shared || hasString(s.Tools, "Shared")
		cursor = cursor || hasString(s.Tools, "Cursor")
	}
	if !shared || !cursor || !c.Discovery.Completed || c.Discovery.Directories == 0 {
		t.Fatalf("missing provenance/report: %+v", c)
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
		if !hasString(source.Tools, "Shared") || !hasString(source.Tools, "Codex") || !hasString(source.Tools, "Pi") {
			t.Fatal("shared root lost declared consumers", source.Tools)
		}
	}
	if !found {
		t.Fatal("shared root missing")
	}
}
