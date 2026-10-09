package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func linkedFixture(t *testing.T) (*Library, string, string, Source, Occurrence) {
	t.Helper()
	l, original, _, id := setup(t)
	root := t.TempDir()
	path := filepath.Join(root, "sample-link")
	if err := os.Symlink(original, path); err != nil {
		t.Fatal(err)
	}
	source, err := l.AddSource(root, true)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 1 {
		t.Fatal("linked catalog", catalog, err)
	}
	for _, occurrence := range catalog.Skills[0].Occurrences {
		if occurrence.SourceID == source.ID && occurrence.Removable {
			return l, original, id, source, occurrence
		}
	}
	t.Fatal("writable final link was not exposed")
	return nil, "", "", Source{}, Occurrence{}
}

func TestLinkTrashPreservesTargetAndRestoresLinkAfterRestart(t *testing.T) {
	l, original, id, source, occurrence := linkedFixture(t)
	preview, err := l.PlanLinkTrash(id, source.ID, occurrence.Path)
	if err != nil || !preview.Link || preview.Count != 1 {
		t.Fatal("link preview", preview, err)
	}
	trashDir := t.TempDir()
	l.trash = func(path string) (string, error) {
		destination := filepath.Join(trashDir, filepath.Base(path))
		return destination, os.Rename(path, destination)
	}
	receipt, err := l.TrashLink(id, source.ID, occurrence.Path, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(occurrence.Path); !os.IsNotExist(err) {
		t.Fatal("selected link remained")
	}
	info, err := os.Lstat(receipt.TrashPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("link was dereferenced", info, err)
	}
	data, _ := os.ReadFile(filepath.Join(original, "SKILL.md"))
	if string(data) != fixture {
		t.Fatal("target content changed")
	}
	restarted, err := New(l.dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.UndoTrash(receipt.ID); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(occurrence.Path)
	if err != nil || target != original {
		t.Fatal("link was not restored", target, err)
	}
	if len(restarted.Settings().RecentTrash) != 0 {
		t.Fatal("restored receipt remained")
	}
}

func TestLinkTrashRefusesReplacementAndRevokedSource(t *testing.T) {
	for _, kind := range []string{"regular file", "replaced link", "revoked source"} {
		t.Run(kind, func(t *testing.T) {
			l, original, id, source, occurrence := linkedFixture(t)
			preview, err := l.PlanLinkTrash(id, source.ID, occurrence.Path)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "regular file":
				_ = os.Remove(occurrence.Path)
				_ = os.WriteFile(occurrence.Path, []byte("replacement"), 0600)
			case "replaced link":
				_ = os.Remove(occurrence.Path)
				_ = os.Symlink(original, occurrence.Path)
			case "revoked source":
				if _, err := l.AddSource(source.Path, false); err != nil {
					t.Fatal(err)
				}
			}
			called := false
			l.trash = func(string) (string, error) { called = true; return "", fmt.Errorf("unexpected move") }
			if _, err := l.TrashLink(id, source.ID, occurrence.Path, preview.Token); err == nil || called {
				t.Fatal("changed/revoked entry moved", err)
			}
			data, _ := os.ReadFile(filepath.Join(original, "SKILL.md"))
			if string(data) != fixture {
				t.Fatal("target content changed")
			}
			if _, err := os.Lstat(occurrence.Path); err != nil {
				t.Fatal("selected entry was removed", err)
			}
		})
	}
}

func TestLinkTrashFailureRestoresEntryWithoutTouchingTarget(t *testing.T) {
	l, original, id, source, occurrence := linkedFixture(t)
	preview, err := l.PlanLinkTrash(id, source.ID, occurrence.Path)
	if err != nil {
		t.Fatal(err)
	}
	l.trash = func(string) (string, error) { return "", fmt.Errorf("fixture Trash unavailable") }
	if _, err := l.TrashLink(id, source.ID, occurrence.Path, preview.Token); err == nil {
		t.Fatal("failure reported success")
	}
	target, err := os.Readlink(occurrence.Path)
	if err != nil || target != original {
		t.Fatal("link was not restored", target, err)
	}
	data, _ := os.ReadFile(filepath.Join(original, "SKILL.md"))
	if string(data) != fixture {
		t.Fatal("target content changed")
	}
}

func TestResourceLinkTrashDoesNotReadOrChangeItsOutsideTarget(t *testing.T) {
	l, skillDir, _, id := setup(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("Outside target."), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(skillDir, "outside-link")
	if err := os.Symlink(outside, entry); err != nil {
		t.Fatal(err)
	}
	preview, err := l.PlanTrash(id, "outside-link")
	if err != nil || !preview.Link || preview.LinkTarget != outside {
		t.Fatal("resource link preview", preview, err)
	}
	trashDir := t.TempDir()
	l.trash = func(path string) (string, error) {
		destination := filepath.Join(trashDir, filepath.Base(path))
		return destination, os.Rename(path, destination)
	}
	receipt, err := l.Trash(id, "outside-link", preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(outside)
	if string(data) != "Outside target." {
		t.Fatal("outside target changed")
	}
	if _, err := os.Lstat(entry); !os.IsNotExist(err) {
		t.Fatal("resource link remained")
	}
	if err := l.UndoTrash(receipt.ID); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(entry)
	if err != nil || target != outside {
		t.Fatal("resource link was not restored", target, err)
	}
}
