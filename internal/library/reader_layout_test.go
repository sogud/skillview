package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReaderLayoutPersistenceAndValidation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), []byte(`{"theme":"dark"}`), 0600); err != nil {
		t.Fatal(err)
	}
	l, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p := l.Settings(); p.ReaderFontSize != 14 || p.ExplorerWidth != 244 || p.Theme != "dark" {
		t.Fatal("old settings did not retain defaults and appearance", p)
	}
	if err := l.SetReaderLayout(18, 320); err != nil {
		t.Fatal(err)
	}
	restored, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p := restored.Settings(); p.ReaderFontSize != 18 || p.ExplorerWidth != 320 || p.Theme != "dark" {
		t.Fatal("reading preferences lost", p)
	}
	for _, values := range [][2]int{{11, 320}, {21, 320}, {18, 179}, {18, 401}} {
		if err := restored.SetReaderLayout(values[0], values[1]); err == nil {
			t.Fatal("invalid layout accepted", values)
		}
		if p := restored.Settings(); p.ReaderFontSize != 18 || p.ExplorerWidth != 320 {
			t.Fatal("invalid layout changed preferences", p)
		}
	}
}

func TestReaderLayoutSaveFailurePreservesPreferences(t *testing.T) {
	l, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.SetReaderLayout(16, 280); err != nil {
		t.Fatal(err)
	}
	// Use an isolated missing directory to force a save failure.
	l.dataDir = filepath.Join(t.TempDir(), "missing")
	if err := l.SetReaderLayout(20, 400); err == nil {
		t.Fatal("save unexpectedly succeeded")
	}
	if p := l.Settings(); p.ReaderFontSize != 16 || p.ExplorerWidth != 280 {
		t.Fatal("failed save changed reading preferences", p)
	}
}
