package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestKnownScanReportsReadingProgressWithoutBroadDiscovery(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first", "second"} {
		dir := filepath.Join(root, name)
		_ = os.MkdirAll(dir, 0755)
		_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0644)
	}
	l, _ := New(t.TempDir(), nil)
	_, _ = l.AddSource(root, false)
	var progress ScanProgress
	catalog, err := l.Scan(context.Background(), func(p ScanProgress) {
		if p.Phase != "reading" {
			t.Fatal("unexpected discovery", p)
		}
		progress = p
	})
	if err != nil || len(catalog.Skills) != 2 || progress.Skills != 2 || progress.SourcesDone != 1 || progress.SourcesTotal != 1 {
		t.Fatalf("reading progress: %+v %v", progress, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Scan(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled scan continued", err)
	}
}
