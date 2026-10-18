package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestArchiveRoundTripSurvivesRestartAndStaysOutOfScans(t *testing.T) {
	l, dir, source, id := setup(t)
	l.archiveDir = filepath.Join(source.ResolvedPath, ".skillview-archive")
	original, _ := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep all resources"), 0600)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	_ = os.WriteFile(outside, []byte("link target"), 0644)
	_ = os.Symlink(outside, filepath.Join(dir, "linked.txt"))
	preview, err := l.PlanArchive(id)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := l.ArchiveSkill(id, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatal("original remained", err)
	}
	if data, _ := os.ReadFile(filepath.Join(receipt.ArchivedPath, "SKILL.md")); string(data) != string(original) {
		t.Fatal("archive changed original bytes")
	}
	if link, _ := os.Readlink(filepath.Join(receipt.ArchivedPath, "linked.txt")); link != outside {
		t.Fatal("resource link was followed or changed")
	}
	if catalog, err := l.Scan(context.Background()); err != nil || len(catalog.Skills) != 0 {
		t.Fatal("archived Skill was scanned", catalog, err)
	}
	// Explicitly selecting the archive must not reactivate its payloads either.
	if _, err := l.AddSource(l.archiveDir, true); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "saved-for-later")
	if err := os.Symlink(l.archiveDir, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := l.AddSource(alias, true); err != nil {
		t.Fatal(err)
	}
	if catalog, err := l.Scan(context.Background()); err != nil || len(catalog.Skills) != 0 {
		t.Fatal("explicit archive source was scanned", err)
	}
	restarted, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	restarted.archiveDir = l.archiveDir
	archived, err := restarted.ArchivedSkills()
	if err != nil || len(archived.Entries) != 1 || !archived.Entries[0].CanRestore {
		t.Fatal("receipt did not survive restart", archived, err)
	}
	if err := restarted.RestoreArchivedSkill(receipt.ID); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(data) != string(original) {
		t.Fatal("restore changed original bytes")
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "notes.txt")); string(data) != "keep all resources" {
		t.Fatal("resource was lost")
	}
	if archived, err := restarted.ArchivedSkills(); err != nil || len(archived.Entries) != 0 {
		t.Fatal("restored item remained", archived, err)
	}
	if data, _ := os.ReadFile(outside); string(data) != "link target" {
		t.Fatal("outside target changed")
	}
}

func TestArchiveMovesOnlySelectedCopyAndRefusesNestedDestination(t *testing.T) {
	l, dir, _, id := setup(t)
	otherRoot := t.TempDir()
	other := filepath.Join(otherRoot, "sample")
	if err := os.Mkdir(other, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "SKILL.md"), []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := l.AddSource(otherRoot, true); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	l.archiveDir = filepath.Join(dir, "archive")
	if _, err := l.PlanArchive(id); err == nil {
		t.Fatal("archive accepted inside selected Skill")
	}
	l.archiveDir = filepath.Join(t.TempDir(), ".skillview-archive")
	preview, err := l.PlanArchive(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.ArchiveSkill(id, preview.Token); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(other, "SKILL.md")); string(data) != fixture {
		t.Fatal("independent copy was changed")
	}
	catalog, err := l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 1 {
		t.Fatal("remaining copy lost from catalog", catalog, err)
	}
}

func TestArchiveRefusesReadOnlyChangedFilesAndRestoreCollisions(t *testing.T) {
	l, dir, source, id := setup(t)
	l.archiveDir = filepath.Join(t.TempDir(), ".skillview-archive")
	preview, err := l.PlanArchive(id)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "new.txt"), []byte("external change"), 0644)
	if _, err := l.ArchiveSkill(id, preview.Token); err == nil {
		t.Fatal("stale preview was accepted")
	}
	if _, err := l.AddSource(source.Path, false); err != nil {
		t.Fatal(err)
	}
	if _, err := l.PlanArchive(id); err == nil {
		t.Fatal("read-only source was archived")
	}
	if _, err := l.AddSource(source.Path, true); err != nil {
		t.Fatal(err)
	}
	preview, _ = l.PlanArchive(id)
	receipt, err := l.ArchiveSkill(id, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Mkdir(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("new Skill"), 0644)
	if err := l.RestoreArchivedSkill(receipt.ID); err == nil {
		t.Fatal("restore overwrote occupied original")
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(data) != "new Skill" {
		t.Fatal("new copy changed")
	}
	if _, err := os.Stat(filepath.Join(receipt.ArchivedPath, "SKILL.md")); err != nil {
		t.Fatal("archive lost after collision", err)
	}
	if err := l.RestoreArchivedSkill("../escape"); err == nil {
		t.Fatal("invalid archive id accepted")
	}
}

func TestArchiveFailureRollsBackAndInterruptedStageCanRecover(t *testing.T) {
	l, dir, _, id := setup(t)
	l.archiveDir = filepath.Join(t.TempDir(), ".skillview-archive")
	preview, _ := l.PlanArchive(id)
	l.archiveMover = func(string, string) error { return fmt.Errorf("synthetic disk failure") }
	if _, err := l.ArchiveSkill(id, preview.Token); err == nil {
		t.Fatal("move failure was hidden")
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatal("failed move lost original", err)
	}
	l.archiveMover = nil
	preview, _ = l.PlanArchive(id)
	receipt, err := l.ArchiveSkill(id, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an interrupted cross-volume copy: the complete original is staged,
	// while an incomplete destination already exists.
	metadata := filepath.Join(filepath.Dir(receipt.ArchivedPath), "receipt.json")
	data, _ := os.ReadFile(metadata)
	var record archiveRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(filepath.Dir(dir), record.Stage)
	if err := os.Rename(receipt.ArchivedPath, stage); err != nil {
		t.Fatal(err)
	}
	_ = os.Mkdir(receipt.ArchivedPath, 0700)
	_ = os.WriteFile(filepath.Join(receipt.ArchivedPath, "partial.txt"), []byte("partial"), 0600)
	record.State, record.Identity = "pending", ""
	data, _ = json.Marshal(record)
	_ = os.WriteFile(metadata, data, 0600)
	if err := l.RestoreArchivedSkill(record.ID); err != nil {
		t.Fatal("interrupted original was not recovered", err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(data) != fixture {
		t.Fatal("partial copy restored instead of original")
	}
	if _, err := os.Stat(filepath.Join(receipt.ArchivedPath, "partial.txt")); err != nil {
		t.Fatal("partial recovery evidence deleted")
	}
}

func TestArchiveRefusesReplacedPayloadParentAndArchiveLinks(t *testing.T) {
	for _, replacement := range []string{"payload", "parent", "archive-link"} {
		t.Run(replacement, func(t *testing.T) {
			l, dir, _, id := setup(t)
			l.archiveDir = filepath.Join(t.TempDir(), ".skillview-archive")
			preview, _ := l.PlanArchive(id)
			receipt, err := l.ArchiveSkill(id, preview.Token)
			if err != nil {
				t.Fatal(err)
			}
			switch replacement {
			case "payload":
				_ = os.Rename(receipt.ArchivedPath, receipt.ArchivedPath+"-kept")
				_ = os.Mkdir(receipt.ArchivedPath, 0700)
			case "parent":
				_ = os.Rename(filepath.Dir(dir), filepath.Dir(dir)+"-kept")
				_ = os.Mkdir(filepath.Dir(dir), 0700)
			case "archive-link":
				_ = os.Rename(l.archiveDir, l.archiveDir+"-kept")
				_ = os.Symlink(l.archiveDir+"-kept", l.archiveDir)
			}
			if err := l.RestoreArchivedSkill(receipt.ID); err == nil {
				t.Fatal("replacement was followed")
			}
		})
	}
}

func TestArchiveAcrossVolumes(t *testing.T) {
	root := os.Getenv("SKILLVIEW_ARCHIVE_TEST_VOLUME")
	if root == "" {
		t.Skip("set a disposable directory on a second volume")
	}
	source, err := os.MkdirTemp(root, "cross-volume-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(source) })
	archive := filepath.Join(t.TempDir(), ".skillview-archive")
	var a, b unix.Stat_t
	_ = unix.Stat(source, &a)
	_ = unix.Stat(filepath.Dir(archive), &b)
	if a.Dev == b.Dev {
		t.Skip("fixture directories are on the same volume")
	}
	dir := filepath.Join(source, "sample")
	_ = os.Mkdir(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fixture), 0640)
	_ = os.Symlink("SKILL.md", filepath.Join(dir, "relative-link"))
	l, _ := New(t.TempDir(), nil)
	l.archiveDir = archive
	if _, err := l.AddSource(source, true); err != nil {
		t.Fatal(err)
	}
	catalog, err := l.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	id := catalog.Skills[0].ID
	preview, _ := l.PlanArchive(id)
	receipt, err := l.ArchiveSkill(id, preview.Token)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.RestoreArchivedSkill(receipt.ID); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(data) != fixture {
		t.Fatal("cross-volume bytes changed")
	}
	if link, _ := os.Readlink(filepath.Join(dir, "relative-link")); link != "SKILL.md" {
		t.Fatal("cross-volume link changed")
	}
	info, _ := os.Stat(filepath.Join(dir, "SKILL.md"))
	if info.Mode().Perm() != 0640 {
		t.Fatal("cross-volume permissions changed")
	}
}
