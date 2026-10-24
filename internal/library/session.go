package library

import (
	"path/filepath"
	"skillview/internal/cache"
)

type Session struct {
	Library    *Library
	Cache      *cache.Store
	CacheError error
}

// App and CLI use the same discovery, fixture isolation and catalog store.
func Open(dataDir, home, fixture string, editable bool) (*Session, error) {
	candidates := []Source{}
	if fixture == "" {
		candidates = DefaultSources(home)
	}
	lib, err := New(dataDir, candidates)
	if err != nil {
		return nil, err
	}
	if fixture != "" {
		lib.SetArchiveDirectory(filepath.Join(dataDir, "archive"))
		if _, err = lib.AddSource(fixture, editable); err != nil {
			return nil, err
		}
	}
	store, err := cache.Open(dataDir)
	if err != nil {
		return &Session{Library: lib, CacheError: err}, nil
	}
	lib.SetCache(store)
	return &Session{Library: lib, Cache: store}, nil
}

func (s *Session) Close() error {
	if s.Cache == nil {
		return nil
	}
	return s.Cache.Close()
}
