package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"skillview/internal/cache"
	"sync"
)

type Library struct {
	mu         sync.Mutex
	dataDir    string
	prefs      Preferences
	catalog    Catalog
	refs       map[string]fileRef
	discovery  DiscoveryReport
	cache      *cache.Store
	watchPaths map[string]bool
	trash      func(string) (string, error)
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
	l := &Library{dataDir: dataDir, refs: map[string]fileRef{}, prefs: Preferences{Favorites: map[string]bool{}, Theme: "system", Language: "system"}}
	data, err := os.ReadFile(filepath.Join(dataDir, "preferences.json"))
	if err == nil {
		if err := json.Unmarshal(data, &l.prefs); err != nil {
			return nil, fmt.Errorf("preferences could not be read; keep the file and repair it: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if l.prefs.Favorites == nil {
		l.prefs.Favorites = map[string]bool{}
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
	l.mergeAutomatic(candidates)
	return l, nil
}

func digest(data []byte) string   { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
func objectID(path string) string { return digest([]byte(path))[:24] }

// AddSource is called only after the native directory picker returns a selection.
func (l *Library) AddSource(path string, writable bool) (Source, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
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

func (l *Library) SetFavorite(id string, favorite bool) error {
	return l.SetFavorites([]string{id}, favorite)
}

func (l *Library) SetFavorites(ids []string, favorite bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	previous := make(map[string]bool, len(ids))
	for _, id := range ids {
		if _, ok := l.refs[id]; !ok {
			return fmt.Errorf("Skill no longer exists; rescan the library")
		}
		previous[id] = l.prefs.Favorites[id]
	}
	for _, id := range ids {
		l.prefs.Favorites[id] = favorite
	}
	if err := l.persist(); err != nil {
		for id, value := range previous {
			l.prefs.Favorites[id] = value
		}
		return err
	}
	return nil
}

func (l *Library) Settings() Preferences {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, _ := json.Marshal(l.prefs)
	var p Preferences
	_ = json.Unmarshal(data, &p)
	return p
}
func (l *Library) SetTheme(theme string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
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
	return os.Rename(temp.Name(), filepath.Join(l.dataDir, "preferences.json"))
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
