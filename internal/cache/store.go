// Package cache stores derived catalog and translation data, never source files or keys.
package cache

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"path/filepath"
)

type Store struct{ db *sql.DB }

func Open(dataDir string) (*Store, error) {
	path := filepath.Join(dataDir, "cache.sqlite")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	file.Close()
	uri := url.URL{Scheme: "file", Path: path}
	db, err := sql.Open("sqlite3", uri.String()+"?_journal_mode=WAL&_busy_timeout=2000&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS entries (kind TEXT NOT NULL, key TEXT NOT NULL, value BLOB NOT NULL, updated INTEGER NOT NULL, PRIMARY KEY(kind,key))`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Get(kind, key string) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow("SELECT value FROM entries WHERE kind=? AND key=?", kind, key).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return data, err
}
func (s *Store) Put(kind, key string, data []byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO entries VALUES (?,?,?,unixepoch()) ON CONFLICT(kind,key) DO UPDATE SET value=excluded.value,updated=excluded.updated", kind, key, data); err != nil {
		return err
	}
	if kind == "translation" {
		if _, err = tx.Exec("DELETE FROM entries WHERE kind='translation' AND key NOT IN (SELECT key FROM entries WHERE kind='translation' ORDER BY updated DESC LIMIT 500)"); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) Clear() error { _, err := s.db.Exec("DELETE FROM entries"); return err }
func (s *Store) Close() error { return s.db.Close() }
