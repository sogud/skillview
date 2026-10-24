package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Call with mu held. All app and CLI writers serialize through this same file.
func (l *Library) beginWrite() (func(), error) {
	fd, err := unix.Open(filepath.Join(l.dataDir, "library.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("Another SkillView operation is running. Try again.")
	}
	release := func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = unix.Close(fd) }
	if err = l.reloadPreferences(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (l *Library) reloadPreferences() error {
	data, err := os.ReadFile(filepath.Join(l.dataDir, "preferences.json"))
	if errors.Is(err, os.ErrNotExist) {
		if _, directoryErr := os.Stat(l.dataDir); directoryErr != nil {
			return directoryErr
		}
		if l.preferencesRevision != digest(nil) {
			l.prefs = Preferences{Theme: "system", Language: "system", ReaderFontSize: 14, ExplorerWidth: 244, ExcludedSources: map[string]bool{}}
			l.preferencesRevision = digest(nil)
			l.mergeAutomatic(l.candidates)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if digest(data) == l.preferencesRevision {
		return nil
	}
	prefs := Preferences{Theme: "system", Language: "system", ReaderFontSize: 14, ExplorerWidth: 244}
	if err = json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("Preferences changed and could not be read. Keep the file for recovery.")
	}
	if prefs.ExcludedSources == nil {
		prefs.ExcludedSources = map[string]bool{}
	}
	l.prefs = prefs
	l.preferencesRevision = digest(data)
	l.mergeAutomatic(l.candidates)
	return nil
}
