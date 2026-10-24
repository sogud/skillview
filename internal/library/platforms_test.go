package library

import (
	"path/filepath"
	"testing"
)

func TestPlatformsUseRegistryAndEnvironment(t *testing.T) {
	home := t.TempDir()
	claudeHome := filepath.Join(home, "custom-claude")
	t.Setenv("CLAUDE_CONFIG_DIR", claudeHome)
	t.Setenv("CODEX_HOME", filepath.Join(home, "custom-codex"))
	seen := map[string]bool{}
	for _, platform := range Platforms(home) {
		if platform.ID == "" || seen[platform.ID] {
			t.Fatalf("missing or duplicate Agent ID: %+v", platform)
		}
		seen[platform.ID] = true
	}
	claude, err := FindPlatform(home, "claude")
	if err != nil || claude.WritePath != filepath.Join(claudeHome, "skills") {
		t.Fatalf("ignored Claude configuration: %+v %v", claude, err)
	}
	codex, err := FindPlatform(home, "codex")
	if err != nil || codex.WritePath != filepath.Join(home, ".agents", "skills") {
		t.Fatalf("incorrect Codex write destination: %+v %v", codex, err)
	}
	if !hasString(codex.Paths, filepath.Join(home, "custom-codex", "skills")) {
		t.Fatal("Codex legacy reading path missing")
	}
}
