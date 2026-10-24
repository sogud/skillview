package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"skillview/internal/cache"
	"strings"
	"sync"
)

type Library struct {
	mu                  sync.Mutex
	dataDir             string
	prefs               Preferences
	catalog             Catalog
	refs                map[string]fileRef
	plugins             map[string]Source
	discovery           DiscoveryReport
	cache               *cache.Store
	watchPaths          map[string]bool
	trash               func(string) (string, error)
	archiveDir          string
	archiveMover        func(string, string) error
	preferencesRevision string
	candidates          []Source
}

type fileRef struct {
	sourceID    string
	relativeDir string
	identity    string
	linkedRoot  *Source
	fileName    string
}

func New(dataDir string, candidates []Source) (*Library, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	l := &Library{dataDir: dataDir, refs: map[string]fileRef{}, prefs: Preferences{Theme: "system", Language: "system", ReaderFontSize: 14, ExplorerWidth: 244}}
	data, err := os.ReadFile(filepath.Join(dataDir, "preferences.json"))
	if err == nil {
		if err := json.Unmarshal(data, &l.prefs); err != nil {
			return nil, fmt.Errorf("preferences could not be read; keep the file and repair it: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if l.prefs.ExcludedSources == nil {
		l.prefs.ExcludedSources = map[string]bool{}
	}
	if l.prefs.Language == "" {
		l.prefs.Language = "system"
	}
	for i := range l.prefs.Sources {
		for j, tool := range l.prefs.Sources[i].Tools {
			if tool == "Custom" {
				l.prefs.Sources[i].Tools[j] = "Local"
			}
		}
	}
	l.preferencesRevision = digest(data)
	l.candidates = candidates
	l.mergeAutomatic(candidates)
	return l, nil
}

func digest(data []byte) string   { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
func objectID(path string) string { return digest([]byte(path))[:24] }

// AddSource records an explicit folder selection in the app or CLI.
func (l *Library) AddSource(path string, writable bool) (Source, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return Source{}, err
	}
	defer release()
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Source{}, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return Source{}, err
	}
	identity, err := pathIdentity(resolved)
	if err != nil {
		return Source{}, err
	}
	source := Source{ID: objectID(absolute), Name: filepath.Base(absolute), Path: absolute, ResolvedPath: resolved, Identity: identity, Approved: true, Writable: writable, Tools: []string{"Local"}}
	previous := append([]Source(nil), l.prefs.Sources...)
	replaced := false
	for i, existing := range l.prefs.Sources {
		if existing.ID == source.ID {
			source.Name = existing.Name
			source.Tools = existing.Tools
			l.prefs.Sources[i] = source
			replaced = true
			break
		}
	}
	if !replaced {
		l.prefs.Sources = append(l.prefs.Sources, source)
	}
	wasExcluded := l.prefs.ExcludedSources[source.ID]
	delete(l.prefs.ExcludedSources, source.ID)
	if err := l.persist(); err != nil {
		l.prefs.Sources = previous
		if wasExcluded {
			l.prefs.ExcludedSources[source.ID] = true
		}
		return Source{}, err
	}
	return source, nil
}

func (l *Library) RemoveSource(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return err
	}
	defer release()
	previous := append([]Source(nil), l.prefs.Sources...)
	sources := []Source{}
	for _, source := range previous {
		if source.ID != id {
			sources = append(sources, source)
		}
	}
	l.prefs.Sources = sources
	wasExcluded := l.prefs.ExcludedSources[id]
	l.prefs.ExcludedSources[id] = true
	if err := l.persist(); err != nil {
		l.prefs.Sources = previous
		if !wasExcluded {
			delete(l.prefs.ExcludedSources, id)
		}
		return err
	}
	return nil
}

func (l *Library) SetLanguage(language string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return err
	}
	defer release()
	if language != "system" && language != "en" && language != "zh-CN" {
		return fmt.Errorf("unknown interface language")
	}
	previous := l.prefs.Language
	l.prefs.Language = language
	if err := l.persist(); err != nil {
		l.prefs.Language = previous
		return err
	}
	return nil
}

func (l *Library) Settings() Preferences {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.reloadPreferences()
	data, _ := json.Marshal(l.prefs)
	var p Preferences
	_ = json.Unmarshal(data, &p)
	return p
}
func (l *Library) SetTheme(theme string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return err
	}
	defer release()
	if theme != "system" && theme != "light" && theme != "dark" {
		return fmt.Errorf("unknown appearance")
	}
	previous := l.prefs.Theme
	l.prefs.Theme = theme
	if err := l.persist(); err != nil {
		l.prefs.Theme = previous
		return err
	}
	return nil
}

func (l *Library) SetReaderLayout(fontSize, explorerWidth int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return err
	}
	defer release()
	if fontSize < 12 || fontSize > 20 || explorerWidth < 180 || explorerWidth > 400 {
		return fmt.Errorf("invalid reading layout")
	}
	previousFontSize, previousWidth := l.prefs.ReaderFontSize, l.prefs.ExplorerWidth
	l.prefs.ReaderFontSize, l.prefs.ExplorerWidth = fontSize, explorerWidth
	if err := l.persist(); err != nil {
		l.prefs.ReaderFontSize, l.prefs.ExplorerWidth = previousFontSize, previousWidth
		return err
	}
	return nil
}

func (l *Library) SetAccentColor(color string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	release, err := l.beginWrite()
	if err != nil {
		return err
	}
	defer release()
	if color != "" && !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(color) {
		return fmt.Errorf("invalid theme color")
	}
	previous := l.prefs.AccentColor
	l.prefs.AccentColor = strings.ToLower(color)
	if err := l.persist(); err != nil {
		l.prefs.AccentColor = previous
		return err
	}
	return nil
}

func (l *Library) persist() error {
	data, err := json.MarshalIndent(l.prefs, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(l.dataDir, ".preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	if _, err = temp.Write(data); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Rename(temp.Name(), filepath.Join(l.dataDir, "preferences.json")); err != nil {
		return err
	}
	l.preferencesRevision = digest(data)
	return nil
}

func (l *Library) source(id string) (Source, error) {
	for _, source := range l.prefs.Sources {
		if source.ID == id {
			return source, nil
		}
	}
	return Source{}, fmt.Errorf("source was removed; select a directory again")
}

func (l *Library) SetCache(store *cache.Store) { l.mu.Lock(); defer l.mu.Unlock(); l.cache = store }

func (l *Library) WatchDirectories() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	paths := []string{}
	for path := range l.watchPaths {
		paths = append(paths, path)
	}
	return paths
}
