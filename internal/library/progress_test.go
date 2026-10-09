package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryProgressHandlesOverlappingAndLinkedRoots(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "project", "custom", "sample")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	l, _ := New(t.TempDir(), nil)
	last := 0
	err := l.discover(context.Background(), []string{alias, root, filepath.Join(root, "project")}, root, nil, func(p ScanProgress) {
		if p.Phase != "discovering" || p.Directories <= last || p.Root == "" {
			t.Fatalf("progress: %+v", p)
		}
		last = p.Directories
	})
	if err != nil || last != 4 {
		t.Fatalf("walk duplicated overlapping roots: %d %v", last, err)
	}
	var progress ScanProgress
	catalog, err := l.Scan(context.Background(), func(p ScanProgress) { progress = p })
	if err != nil || len(catalog.Skills) != 1 || progress.Skills != 1 || progress.SourcesDone != progress.SourcesTotal {
		t.Fatalf("reading progress: %+v %v", progress, err)
	}
}

func TestCanceledDiscoveryDoesNotClaimCompletion(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, "child"), 0755)
	l, _ := New(t.TempDir(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	err := l.discover(ctx, []string{root}, root, nil, func(ScanProgress) { cancel() })
	if !errors.Is(err, context.Canceled) || l.discovery.Completed || len(l.Settings().Sources) != 0 {
		t.Fatal("canceled discovery claimed completion", err)
	}
}
