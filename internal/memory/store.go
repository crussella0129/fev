package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register driver
)

// Store is the persistent memory database.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at the given path.
// It creates parent directories and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates tables if they don't exist.
func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS facts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			claim TEXT NOT NULL,
			source_file TEXT,
			source_line INTEGER,
			created_at INTEGER DEFAULT (strftime('%s','now')),
			verified_at INTEGER DEFAULT (strftime('%s','now'))
		)`,
		`CREATE TABLE IF NOT EXISTS hunches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			belief TEXT NOT NULL,
			confidence REAL DEFAULT 0.5,
			created_at INTEGER DEFAULT (strftime('%s','now'))
		)`,
		`CREATE TABLE IF NOT EXISTS corrections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wrong_claim TEXT NOT NULL,
			right_claim TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s','now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_source ON facts(source_file)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_claim ON facts(claim)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}
