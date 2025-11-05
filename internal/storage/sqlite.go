package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLite creates a new SQLite storage backend
// For production: uses /tmp/gcp-visualizer/cache.db
// For testing: use ":memory:" as dbPath
func NewSQLite(dbPath string) (*SQLiteStorage, error) {
	// Create directory for file-based databases
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Set pragmas for performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, err
	}

	s := &SQLiteStorage{db: db}
	return s, s.migrate()
}

// NewDefaultSQLite creates storage in /tmp/gcp-visualizer/
func NewDefaultSQLite() (*SQLiteStorage, error) {
	dbPath := filepath.Join("/tmp", "gcp-visualizer", "cache.db")
	return NewSQLite(dbPath)
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
