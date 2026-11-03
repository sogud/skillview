package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStandaloneCommandInstallation(t *testing.T) {
	home := t.TempDir()
	exe := filepath.Join(home, "downloads", "skillview")
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
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(path); err != nil || target != resolved {
		t.Fatalf("unexpected command link: %s, %v", target, err)
	}
	if _, err := InstallCommand(path, home); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("standalone"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallCommand(path, home); err != nil {
		t.Fatalf("installed standalone binary was not idempotent: %v", err)
	}
}

func TestCommandInstallationRejectsInvalidExecutable(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "not-executable")
	if err := os.WriteFile(file, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{file, home, filepath.Join(home, "missing")} {
		if _, err := InstallCommand(path, home); err == nil {
			t.Fatalf("accepted invalid executable: %s", path)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatal("invalid executable created command directory")
	}
}

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
