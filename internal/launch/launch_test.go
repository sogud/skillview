package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchCommandInstallationAndConflict(t *testing.T) {
	home := t.TempDir()
	exe := filepath.Join(home, "SkillView.app", "Contents", "MacOS", "SkillView")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("fixture"), 0755); err != nil {
		t.Fatal(err)
	}
	path, err := InstallCommand(exe, home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = InstallCommand(exe, home); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("keep"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err = InstallCommand(exe, home); err == nil {
		t.Fatal("overwrote another command")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep" {
		t.Fatal("command changed")
	}
}
