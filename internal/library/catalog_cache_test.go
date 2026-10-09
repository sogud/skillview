package library

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"skillview/internal/cache"
	"strings"
	"testing"
)

func TestCachedCatalogOpensCurrentFilesWithoutScanning(t *testing.T) {
	l, original, _, id := setup(t)
	store, err := cache.Open(l.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	l.SetCache(store)
	if _, err := l.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(original, "SKILL.md"), []byte(fixture+"Current disk text."), 0644); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(l.dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	restarted.SetCache(store)
	catalog, err := restarted.CachedCatalog()
	if err != nil || catalog.NeedsScan || len(catalog.Skills) != 1 {
		t.Fatal("saved catalog not restored", err)
	}
	if len(restarted.WatchDirectories()) != 0 {
		t.Fatal("cached startup traversed folders")
	}
	doc, err := restarted.Read(id, "SKILL.md")
	if err != nil || !strings.Contains(doc.Text, "Current disk text.") {
		t.Fatal("cached bytes replaced original", err)
	}
	if err := os.Rename(original, original+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(original, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(original, "SKILL.md"), []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Read(id, "SKILL.md"); err == nil {
		t.Fatal("cached identity accepted a replaced folder")
	}
}

func TestCachedCatalogCannotRestoreRemovedSourceOrGrantLinkedAccess(t *testing.T) {
	l, _, source, id := setup(t)
	store, err := cache.Open(l.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	l.SetCache(store)
	if _, err := l.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, _ := store.Get("catalog", "current")
	var snapshot catalogSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	ref := snapshot.References[id]
	outside := t.TempDir()
	outside, err = filepath.EvalSymlinks(outside)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte("outside original"), 0644); err != nil {
		t.Fatal(err)
	}
	identity, err := pathIdentity(outside)
	if err != nil {
		t.Fatal(err)
	}
	ref.LinkedRoot = &Source{ID: source.ID, Path: outside, ResolvedPath: outside, Identity: identity, Approved: true, Writable: true}
	ref.RelativeDir, ref.Identity = "", identity
	snapshot.References[id] = ref
	snapshot.Skills[0].Path = outside
	snapshot.Skills[0].ID = objectID(outside)
	snapshot.References[objectID(outside)] = ref
	delete(snapshot.References, id)
	data, _ = json.Marshal(snapshot)
	if err := store.Put("catalog", "current", data); err != nil {
		t.Fatal(err)
	}
	restarted, _ := New(l.dataDir, nil)
	restarted.SetCache(store)
	if catalog, err := restarted.CachedCatalog(); err != nil || catalog.NeedsScan || len(restarted.refs) != 1 {
		t.Fatal("synthetic snapshot was not restored", err)
	}
	if _, err := restarted.Read(objectID(outside), "SKILL.md"); err == nil || !strings.Contains(err.Error(), "Skill link changed") {
		t.Fatal("cached linked root granted outside access", err)
	}
	if err := restarted.RemoveSource(source.ID); err != nil {
		t.Fatal(err)
	}
	catalog, err := restarted.CachedCatalog()
	if err != nil || len(catalog.Skills) != 0 || !catalog.NeedsScan {
		t.Fatal("removed source returned through cache", err)
	}
}

func TestCachedAutomaticLinkChecksCurrentTargetAndStaysReadOnly(t *testing.T) {
	root, original := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(original, "SKILL.md"), []byte(fixture), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "sample")
	if err := os.Symlink(original, link); err != nil {
		t.Fatal(err)
	}
	source, ok := automaticSource(root, "fixture", "global", []string{"fixture"})
	if !ok {
		t.Fatal("fixture source missing")
	}
	l, _ := New(t.TempDir(), []Source{source})
	if err := l.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	store, err := cache.Open(l.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	l.SetCache(store)
	catalog, err := l.Scan(context.Background())
	if err != nil || len(catalog.Skills) != 1 {
		t.Fatal("linked fixture scan failed", err)
	}
	id := catalog.Skills[0].ID
	restarted, _ := New(l.dataDir, nil)
	restarted.SetCache(store)
	if _, err := restarted.CachedCatalog(); err != nil {
		t.Fatal(err)
	}
	doc, err := restarted.Read(id, "SKILL.md")
	if err != nil || doc.Editable || doc.Text != fixture {
		t.Fatal("cached linked read failed", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Read(id, "SKILL.md"); err == nil {
		t.Fatal("changed cached link was followed")
	}
}

func TestLegacyCatalogRequiresOneManualScan(t *testing.T) {
	l, _, _, _ := setup(t)
	store, err := cache.Open(l.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	data, _ := json.Marshal(l.catalog)
	if err := store.Put("catalog", "current", data); err != nil {
		t.Fatal(err)
	}
	restarted, _ := New(l.dataDir, nil)
	restarted.SetCache(store)
	catalog, err := restarted.CachedCatalog()
	if err != nil || !catalog.NeedsScan || len(catalog.Skills) != 1 {
		t.Fatal("legacy cache did not request a scan", err)
	}
	if _, err := restarted.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if catalog, err = restarted.CachedCatalog(); err != nil || catalog.NeedsScan {
		t.Fatal("manual scan did not enable cached reading", err)
	}
}
