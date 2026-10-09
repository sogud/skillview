package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestObserveAtomicReplacementAndNewDirectories(t *testing.T) {
	root := t.TempDir()
	events := make(chan struct{}, 10)
	watcher, err := New(func() {
		select {
		case events <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	if !watcher.Set([]string{root}) {
		t.Fatal("directory watch failed")
	}
	target := filepath.Join(root, "SKILL.md")
	_ = os.WriteFile(target, []byte("Original"), 0644)
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("creation not observed")
	}
	staged := filepath.Join(root, "temporary")
	_ = os.WriteFile(staged, []byte("Changed"), 0644)
	_ = os.Rename(staged, target)
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("atomic replacement not observed")
	}
	child := filepath.Join(root, "references")
	_ = os.Mkdir(child, 0755)
	if !watcher.Set([]string{root, child}) {
		t.Fatal("new directory not registered")
	}
	_ = os.WriteFile(filepath.Join(child, "guide.md"), []byte("Guide"), 0644)
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("nested resource not observed")
	}
	watcher.Set(nil)
}
