package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"skillview/internal/library"
)

func TestManualScanReadsOnlyKnownAndExplicitFolders(t *testing.T) {
	known := t.TempDir()
	search := t.TempDir()
	for _, root := range []string{known, search} {
		skill := filepath.Join(root, "sample")
		if err := os.Mkdir(skill, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: sample\ndescription: Disposable fixture.\n---\n# Sample\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	l, _ := library.New(t.TempDir(), nil)
	if _, err := l.AddSource(known, false); err != nil {
		t.Fatal(err)
	}
	s := New(l, Native{KnownSources: func() []library.Source { return nil }}, nil)
	catalog, err := s.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 1 || catalog.Discovery.Directories != 0 {
		t.Fatalf("initial catalog: %d skills, %d discovery directories, %v", len(catalog.Skills), catalog.Discovery.Directories, err)
	}
	if _, err := s.Read(catalog.Skills[0].ID, "SKILL.md"); err != nil {
		t.Fatal("known Skill was not readable", err)
	}
}
