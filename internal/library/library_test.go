package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

const fixture = "---\nname: sample\ndescription: Read a sample Skill.\n---\n\n# Sample\n\nOriginal text.\n"

func setup(t *testing.T) (*Library, string, Source, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "sample")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	l, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	source, err := l.AddSource(root, true)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 {
		t.Fatalf("catalog: %+v", catalog)
	}
	return l, dir, source, catalog.Skills[0].ID
}

func TestCreatePublishesCompleteSkillAndNeverOverwrites(t *testing.T) {
	l, _, source, _ := setup(t)
	id, err := l.Create(source.ID, "new-skill", "A description: with quotes \"and\" a newline.\nSecond line.")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source.ResolvedPath, "new-skill", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("published SKILL.md missing:", err)
	}
	name, desc, warnings := parseSkill(data, filepath.Dir(path))
	if name != "new-skill" || !strings.Contains(desc, "Second line.") || len(warnings) != 0 {
		t.Fatalf("invalid created Skill: %s %s %v", name, desc, warnings)
	}
	if _, err := l.Create(source.ID, "new-skill", "replacement"); err == nil {
		t.Fatal("existing directory replaced")
	}
	unchanged, _ := os.ReadFile(path)
	if string(unchanged) != string(data) {
		t.Fatal("duplicate creation modified the file")
	}
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 2 || id == "" {
		t.Fatal("created Skill not discoverable")
	}
	entries, _ := os.ReadDir(source.ResolvedPath)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".skillview-create-") {
			t.Fatal("temporary creation left behind")
		}
	}
	for _, name := range []string{"../escape", "Bad", "a--b", "-bad", "a/b", ""} {
		if _, err := l.Create(source.ID, name, "description"); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestCreateRejectsSingleSkillSource(t *testing.T) {
	l, dir, _, _ := setup(t)
	source, err := l.AddSource(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Create(source.ID, "nested", "description"); err == nil {
		t.Fatal("nested Skill created as a resource")
	}
	if _, err := os.Stat(filepath.Join(dir, "nested")); !os.IsNotExist(err) {
		t.Fatal("single Skill source changed")
	}
}

func TestSavePreservesRawFrontmatterCRLFPermissionsAndAttributes(t *testing.T) {
	l, dir, _, id := setup(t)
	path := filepath.Join(dir, "SKILL.md")
	original := "---\r\n# Keep this comment\r\nname: sample\r\ndescription: A sample.\r\ncustom-field: 'keep: exactly'\r\n---\r\n\r\n# Original\r\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := unix.Setxattr(path, "com.skillview.fixture", []byte("keep"), 0); err != nil {
		t.Fatal(err)
	}
	doc, err := l.Read(id, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Newline != "\r\n" {
		t.Fatal("CRLF not detected")
	}
	draft := strings.Replace(original, "# Original", "# Edited", 1)
	result, err := l.Save(id, "SKILL.md", doc.Revision, draft)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "saved" || result.Document.Text != draft {
		t.Fatal("save returned wrong content")
	}
	saved, _ := os.ReadFile(path)
	if string(saved) != draft {
		t.Fatal("raw bytes changed")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("file permissions changed")
	}
	attribute := make([]byte, 4)
	if _, err := unix.Getxattr(path, "com.skillview.fixture", attribute); err != nil || string(attribute) != "keep" {
		t.Fatal("extended attribute lost", err)
	}
	backup, err := os.ReadFile(result.RecoveryPath)
	if err != nil || string(backup) != original {
		t.Fatal("original recovery copy missing")
	}
	backupInfo, _ := os.Stat(result.RecoveryPath)
	if backupInfo.Mode().Perm() != 0600 {
		t.Fatal("recovery copy is not private")
	}
}

func TestSaveConflictLeavesExternalVersionIntact(t *testing.T) {
	l, dir, _, id := setup(t)
	doc, _ := l.Read(id, "SKILL.md")
	external := fixture + "External change.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(external), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := l.Save(id, "SKILL.md", doc.Revision, fixture+"My draft.\n")
	if err != nil || result.Status != "conflict" || result.Document.Text != external {
		t.Fatalf("conflict not returned: %+v %v", result, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if string(data) != external {
		t.Fatal("external version was overwritten")
	}
}

func TestResourcesRejectTraversalAndSymlinks(t *testing.T) {
	l, dir, _, id := setup(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(target, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "file-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "folder-link")); err != nil {
		t.Fatal(err)
	}
	resources, err := l.Resources(id)
	if err != nil {
		t.Fatal(err)
	}
	linkCount := 0
	for _, r := range resources {
		if r.Kind == "link" {
			linkCount++
		}
	}
	if linkCount != 2 {
		t.Fatal("resource links were followed")
	}
	for _, path := range []string{"", "../secret.txt", "/etc/passwd", "a//b", "./SKILL.md", "file-link", "folder-link/secret.txt"} {
		if _, err := l.Read(id, path); err == nil {
			t.Fatalf("read accepted unsafe path %q", path)
		}
		if _, err := l.Save(id, path, "hash", "overwrite"); err == nil {
			t.Fatalf("save accepted unsafe path %q", path)
		}
	}
	data, _ := os.ReadFile(target)
	if string(data) != "outside" {
		t.Fatal("outside target changed")
	}
}

func TestReadAndSaveRejectReplacedSkillAndResourceParent(t *testing.T) {
	for _, replaceSkill := range []bool{false, true} {
		t.Run(map[bool]string{false: "resource parent", true: "Skill directory"}[replaceSkill], func(t *testing.T) {
			l, dir, _, id := setup(t)
			resourceDir := filepath.Join(dir, "references")
			if err := os.Mkdir(resourceDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(resourceDir, "guide.md"), []byte("Original guide"), 0644); err != nil {
				t.Fatal(err)
			}
			doc, err := l.Read(id, "references/guide.md")
			if err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "guide.md"), []byte("outside"), 0644); err != nil {
				t.Fatal(err)
			}
			replaced := resourceDir
			if replaceSkill {
				replaced = dir
			}
			if err := os.Rename(replaced, replaced+"-old"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, replaced); err != nil {
				t.Fatal(err)
			}
			if _, err := l.Read(id, "references/guide.md"); err == nil {
				t.Fatal("read followed replaced directory")
			}
			if _, err := l.Save(id, "references/guide.md", doc.Revision, "draft"); err == nil {
				t.Fatal("save followed replaced directory")
			}
			data, _ := os.ReadFile(filepath.Join(outside, "guide.md"))
			if string(data) != "outside" {
				t.Fatal("outside file modified")
			}
		})
	}
}

func TestHardLinksAreReadOnly(t *testing.T) {
	l, dir, _, id := setup(t)
	if err := os.Link(filepath.Join(dir, "SKILL.md"), filepath.Join(dir, "shared.md")); err != nil {
		t.Fatal(err)
	}
	doc, err := l.Read(id, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Editable {
		t.Fatal("hard-linked file was editable")
	}
	if _, err := l.Save(id, "SKILL.md", doc.Revision, "draft"); err == nil {
		t.Fatal("hard-linked file was replaced")
	}
}

func TestScannerDeduplicatesPhysicalDirectoriesAndKeepsSources(t *testing.T) {
	l, dir, source, id := setup(t)
	aliasRoot := t.TempDir()
	if err := os.Symlink(dir, filepath.Join(aliasRoot, "alias")); err != nil {
		t.Fatal(err)
	}
	alias, err := l.AddSource(aliasRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 {
		t.Fatalf("duplicate physical Skills: %+v", catalog.Skills)
	}
	skill := catalog.Skills[0]
	if skill.ID != id || len(skill.Occurrences) != 2 || !skill.Editable {
		t.Fatalf("lost occurrence: %+v", skill)
	}
	if skill.Occurrences[0].SourceID != source.ID || skill.Occurrences[1].SourceID != alias.ID || !skill.Occurrences[1].Linked {
		t.Fatal("discovery source was lost")
	}
	// Same name and contents in a different directory must remain independent.
	second := filepath.Join(aliasRoot, "second")
	_ = os.Mkdir(second, 0755)
	_ = os.WriteFile(filepath.Join(second, "SKILL.md"), []byte(fixture), 0644)
	catalog, err = l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 2 {
		t.Fatal("different physical directory collapsed", err)
	}
}

func TestUnauthorizedLinksAndCyclesDoNotBlockOtherSkills(t *testing.T) {
	l, dir, source, _ := setup(t)
	outside := t.TempDir()
	_ = os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte(fixture), 0644)
	_ = os.Symlink(outside, filepath.Join(source.Path, "outside"))
	_ = os.Symlink("cycle", filepath.Join(source.Path, "cycle"))
	_ = os.Symlink(source.ResolvedPath, filepath.Join(source.Path, "back"))
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 || len(catalog.Issues) < 3 {
		t.Fatalf("link handling: %+v", catalog)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if string(data) != fixture {
		t.Fatal("scan changed file")
	}
}

func TestReadOnlySourceAndRootReplacementRefuseWrites(t *testing.T) {
	l, _, source, id := setup(t)
	_, err := l.AddSource(source.Path, false)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = l.Scan(context.Background())
	doc, err := l.Read(id, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Editable {
		t.Fatal("read-only source editable")
	}
	if _, err := l.Save(id, "SKILL.md", doc.Revision, "draft"); err == nil {
		t.Fatal("read-only save allowed")
	}
	if _, err := l.Create(source.ID, "created", "description"); err == nil {
		t.Fatal("read-only creation allowed")
	}
	if err := os.Rename(source.Path, source.Path+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(source.Path, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Read(id, "SKILL.md"); err == nil {
		t.Fatal("replaced root allowed")
	}
}

func TestInvalidYAMLRemainsReadable(t *testing.T) {
	l, dir, _, id := setup(t)
	original := "---\nname: [invalid\n---\n\nBody\n"
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(original), 0644)
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 || len(catalog.Skills[0].Warnings) == 0 {
		t.Fatal("bad YAML disappeared")
	}
	doc, err := l.Read(id, "SKILL.md")
	if err != nil || doc.Text != original {
		t.Fatal("raw broken file not available", err)
	}
}

func TestPreferencesSurviveRestartAndRevocation(t *testing.T) {
	l, _, _, id := setup(t)
	if err := l.SetFavorite(id, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	restored, err := New(l.dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := restored.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 || !catalog.Skills[0].Favorite || restored.Settings().Theme != "dark" {
		t.Fatal("preferences lost")
	}
	sourceID := catalog.Sources[0].ID
	if err := restored.RemoveSource(sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Read(id, "SKILL.md"); err == nil {
		t.Fatal("revoked source readable")
	}
	if err := os.WriteFile(filepath.Join(l.dataDir, "preferences.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(l.dataDir, nil); err == nil {
		t.Fatal("broken preferences were silently replaced")
	}
}
