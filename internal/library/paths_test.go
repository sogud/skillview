package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryTraversalWithoutListingPermission(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "allowed")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(parent, 0311); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(parent, 0700)
	directory, err := openAbsoluteDirectory(canonical)
	if err != nil {
		t.Fatal("traversable parent rejected", err)
	}
	directory.Close()
	alias := filepath.Join(t.TempDir(), "linked")
	if err = os.Symlink(canonical, alias); err != nil {
		t.Fatal(err)
	}
	if linked, err := openAbsoluteDirectory(alias); err == nil {
		linked.Close()
		t.Fatal("symlink traversed")
	}
}
