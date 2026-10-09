package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportCreatesSeparateFileAndRejectsOriginalsAliasesAndStaleContent(t *testing.T) {
	l, skillDir, _, id := setup(t)
	doc, err := l.Read(id, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	destinationDir := t.TempDir()
	destination := filepath.Join(destinationDir, "translated.md")
	if err := l.Export(id, "SKILL.md", doc.Revision, destination, "# 译文\n"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "# 译文\n" {
		t.Fatal("export content", string(data), err)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("export must be private", info, err)
	}
	if err := l.Export(id, "SKILL.md", doc.Revision, destination, "replacement"); err == nil {
		t.Fatal("existing export replaced")
	}
	original := filepath.Join(skillDir, "SKILL.md")
	alias := filepath.Join(destinationDir, "alias.md")
	if err := os.Symlink(original, alias); err != nil {
		t.Fatal(err)
	}
	hardlink := filepath.Join(destinationDir, "hardlink.md")
	if err := os.Link(original, hardlink); err != nil {
		t.Fatal(err)
	}
	parentAlias := filepath.Join(destinationDir, "linked-parent")
	if err := os.Symlink(skillDir, parentAlias); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{original, alias, hardlink, filepath.Join(skillDir, "translated.md"), filepath.Join(parentAlias, "translated.md")} {
		if err := l.Export(id, "SKILL.md", doc.Revision, target, "replacement"); err == nil {
			t.Fatalf("accepted original/alias destination %s", target)
		}
	}
	unchanged, _ := os.ReadFile(original)
	if string(unchanged) != fixture {
		t.Fatal("original changed")
	}
	if err := os.WriteFile(original, []byte(fixture+"Changed."), 0644); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(destinationDir, "stale.md")
	if err := l.Export(id, "SKILL.md", doc.Revision, stale, "stale translation"); err == nil {
		t.Fatal("stale translation exported")
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatal("stale file published")
	}
	entries, _ := os.ReadDir(destinationDir)
	for _, entry := range entries {
		if len(entry.Name()) >= 18 && entry.Name()[:18] == ".skillview-export-" {
			t.Fatal("temporary export left behind")
		}
	}
}
