package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
}

func TestNewStore_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "deep", "memory.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore with nested path: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected db file to exist at %s", dbPath)
	}
}

func TestNewStore_MigratesOnOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memory.db")

	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	s.Close()

	// Reopen — tables should already exist, migration should be idempotent.
	s2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer s2.Close()

	// Verify the tables exist by querying them.
	tables := []string{"facts", "hunches", "corrections"}
	for _, tbl := range tables {
		row := s2.db.QueryRow("SELECT count(*) FROM " + tbl)
		var n int
		if err := row.Scan(&n); err != nil {
			t.Errorf("table %q not accessible after reopen: %v", tbl, err)
		}
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
