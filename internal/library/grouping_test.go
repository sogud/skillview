package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentGroupingKeepsIndependentCopiesAndChangedVersions(t *testing.T) {
	l, _, _, first := setup(t)
	root := t.TempDir()
	for name, text := range map[string]string{"same": strings.ReplaceAll(fixture, "\n", "\r\n"), "changed": fixture + "A different instruction.\n"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := l.AddSource(root, false); err != nil {
		t.Fatal(err)
	}
	catalog, err := l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 3 {
		t.Fatal(catalog, err)
	}
	keys := map[string]int{}
	ids := []string{}
	for _, skill := range catalog.Skills {
		keys[skill.ContentKey]++
		ids = append(ids, skill.ID)
	}
	if len(keys) != 2 {
		t.Fatal("different versions merged", keys)
	}
	if err := l.SetFavorites(ids, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetFavorites([]string{first, "missing"}, false); err == nil {
		t.Fatal("missing copy accepted")
	}
	if !l.Settings().Favorites[first] {
		t.Fatal("failed group action partially changed favorites")
	}
}
