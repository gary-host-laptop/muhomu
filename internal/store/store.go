package store

import (
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Store is the application's persistence boundary. Widgets and HTTP
// handlers depend on Store methods instead of sharing a package-level
// *sql.DB or duplicating domain types.
type Store struct {
	db *sql.DB
}

// Open creates the SQLite connection and applies the idempotent schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
