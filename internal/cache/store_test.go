package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistenceClearAndPrivatePermissions(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Put("translation", "one", []byte("result")); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	value, err := store.Get("translation", "one")
	if err != nil || string(value) != "result" {
		t.Fatal("cache did not survive restart", err)
	}
	info, _ := os.Stat(filepath.Join(root, "cache.sqlite"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("cache permissions")
	}
	if err = store.Clear(); err != nil {
		t.Fatal(err)
	}
	value, err = store.Get("translation", "one")
	if err != nil || value != nil {
		t.Fatal("clear failed", err)
	}
}
