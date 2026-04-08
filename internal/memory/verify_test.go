package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVerify_Valid(t *testing.T) {
	s := newTestStore(t)
	v := NewVerifier(s)

	// Create a real file.
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Insert a fact, then backdate the file's mtime to before verified_at.
	// verified_at defaults to CURRENT_TIMESTAMP on insert — set file mtime to 1 hour ago.
	past := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(file, past, past); err != nil {
		t.Fatal(err)
	}

	_, err := s.InsertFact("main.go is the entry point", file, 1)
	if err != nil {
		t.Fatal(err)
	}

	result, err := v.Verify("main.go is the entry point", file)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Status != VerifyValid {
		t.Errorf("expected VerifyValid, got %d", result.Status)
	}
	if result.Fact == nil {
		t.Error("expected non-nil Fact")
	}
}

func TestVerify_Stale(t *testing.T) {
	s := newTestStore(t)
	v := NewVerifier(s)

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Insert fact (verified_at = now)
	_, err := s.InsertFact("main.go is the entry point", file, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Set file mtime to 1 second in the future relative to verified_at — making it "modified after".
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(file, future, future); err != nil {
		t.Fatal(err)
	}

	result, err := v.Verify("main.go is the entry point", file)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Status != VerifyStale {
		t.Errorf("expected VerifyStale, got %d", result.Status)
	}
	if !result.Stale {
		t.Error("expected Stale=true")
	}
}

func TestVerify_NotFound(t *testing.T) {
	s := newTestStore(t)
	v := NewVerifier(s)

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	_ = os.WriteFile(file, []byte("package main"), 0644)

	result, err := v.Verify("claim that doesn't exist", file)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Status != VerifyNotFound {
		t.Errorf("expected VerifyNotFound, got %d", result.Status)
	}
}
