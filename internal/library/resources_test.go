package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceCreationRenameAndCollision(t *testing.T) {
	l, dir, _, id := setup(t)
	if _, err := l.CreateResource(id, "notes.md", "# A note\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.CreateResource(id, "notes.md", "replace"); err == nil {
		t.Fatal("resource replaced")
	}
	if err := l.ImportResource(id, "sample.bin", []byte{0, 1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	resources, err := l.Resources(id)
	if err != nil {
		t.Fatal(err)
	}
	revision := ""
	for _, r := range resources {
		if r.Path == "notes.md" {
			revision = r.Revision
		}
	}
	if err = l.RenameResource(id, "notes.md", "renamed.md", revision); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "renamed.md"))
	if string(content) != "# A note\n" {
		t.Fatal("rename changed bytes")
	}
	for _, path := range []string{"../escape", "SKILL.md", ".skillview-save-hidden"} {
		if _, err = l.CreateResource(id, path, "text"); err == nil {
			t.Fatal("accepted unsafe resource", path)
		}
	}
	resources, _ = l.Resources(id)
	for _, r := range resources {
		if r.Path == "renamed.md" {
			revision = r.Revision
		}
	}
	if err = l.RenameResource(id, "renamed.md", "sample.bin", revision); err == nil {
		t.Fatal("rename collision replaced binary")
	}
	_ = os.WriteFile(filepath.Join(dir, "renamed.md"), []byte("External change"), 0644)
	if err = l.RenameResource(id, "renamed.md", "next.md", revision); err == nil {
		t.Fatal("external change ignored")
	}
}

func TestTrashFailureRestoresAndChangedFilesRefuse(t *testing.T) {
	l, dir, _, id := setup(t)
	_, _ = l.CreateResource(id, "notes.md", "Original")
	preview, err := l.PlanTrash(id, "notes.md")
	if err != nil {
		t.Fatal(err)
	}
	l.trash = func(string) (string, error) { return "", fmt.Errorf("fixture failure") }
	if _, err = l.Trash(id, "notes.md", preview.Token); err == nil {
		t.Fatal("trash failure missing")
	}
	content, _ := os.ReadFile(filepath.Join(dir, "notes.md"))
	if string(content) != "Original" {
		t.Fatal("failed Trash lost file")
	}
	preview, _ = l.PlanTrash(id, "")
	_ = os.WriteFile(filepath.Join(dir, "notes.md"), []byte("changed"), 0644)
	if _, err = l.Trash(id, "", preview.Token); err == nil {
		t.Fatal("changed directory moved")
	}
	if _, err = l.PlanTrash(id, "SKILL.md"); err == nil {
		t.Fatal("main file trashed as resource")
	}
}

func TestTrashMovesOnlySelectedOriginalAndReportsLinks(t *testing.T) {
	l, dir, source, id := setup(t)
	alias := t.TempDir()
	_ = os.Symlink(dir, filepath.Join(alias, "linked"))
	_, _ = l.AddSource(alias, false)
	_, _ = l.Scan(context.Background())
	preview, err := l.PlanTrash(id, "")
	if err != nil || len(preview.Links) != 1 || preview.Count != 1 {
		t.Fatal("preview", preview, err)
	}
	trash := t.TempDir()
	l.trash = func(path string) (string, error) {
		target := filepath.Join(trash, filepath.Base(path))
		return target, os.Rename(path, target)
	}
	destination, err := l.Trash(id, "", preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(destination.TrashPath, "SKILL.md")); err != nil {
		t.Fatal("Trash content missing", err)
	}
	if _, err = os.Lstat(filepath.Join(alias, "linked")); err != nil {
		t.Fatal("alias removed", err)
	}
	if _, err = os.Stat(filepath.Join(source.Path, "sample")); !os.IsNotExist(err) {
		t.Fatal("original still present")
	}
	if err = l.UndoTrash(destination.ID); err != nil {
		t.Fatal("Undo Trash", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "SKILL.md")); err != nil || string(data) != fixture {
		t.Fatal("restore lost content", err)
	}
	// Read-only and linked-only entries have no destructive action.
	outside := t.TempDir()
	_ = os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte(fixture), 0644)
	readonly, _ := automaticSource(outside, "sample", "local", []string{"Local"})
	other, _ := New(t.TempDir(), []Source{readonly})
	catalog, _ := other.Scan(context.Background())
	if _, err = other.PlanTrash(catalog.Skills[0].ID, ""); err == nil {
		t.Fatal("read-only Trash allowed")
	}
}

func TestOpenCodeStandaloneMarkdownDoesNotExposeSiblingFiles(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "hello.md"), []byte(fixture), 0644)
	_ = os.WriteFile(filepath.Join(root, "other.txt"), []byte("private sibling"), 0644)
	source, _ := automaticSource(root, "OpenCode", "global", []string{"OpenCode"})
	l, _ := New(t.TempDir(), []Source{source})
	c, err := l.Scan(context.Background())
	if err != nil || len(c.Skills) != 1 {
		t.Fatal("flat discovery", err)
	}
	doc, err := l.Read(c.Skills[0].ID, "SKILL.md")
	if err != nil || doc.Path != "hello.md" || doc.Editable {
		t.Fatal("flat read", doc, err)
	}
	if _, err = l.Read(c.Skills[0].ID, "other.txt"); err == nil {
		t.Fatal("flat Skill exposed sibling")
	}
	resources, _ := l.Resources(c.Skills[0].ID)
	if len(resources) != 1 {
		t.Fatal("flat resources", resources)
	}
}
