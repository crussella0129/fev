package memory

import (
	"path/filepath"
	"testing"
)

func TestInsertFact_FindByQuery(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertFact("main.go imports net/http", "main.go", 5)
	if err != nil {
		t.Fatalf("InsertFact: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	facts, err := s.FindFacts("net/http", 10)
	if err != nil {
		t.Fatalf("FindFacts: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Claim != "main.go imports net/http" {
		t.Errorf("unexpected claim: %q", facts[0].Claim)
	}
	if facts[0].SourceFile != "main.go" {
		t.Errorf("unexpected source file: %q", facts[0].SourceFile)
	}
	if facts[0].SourceLine != 5 {
		t.Errorf("unexpected source line: %d", facts[0].SourceLine)
	}
}

func TestFindFactsBySource(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.InsertFact("fact about main.go", "main.go", 1)
	_, _ = s.InsertFact("another fact about main.go", "main.go", 10)
	_, _ = s.InsertFact("fact about other.go", "other.go", 1)

	facts, err := s.FindFactsBySource("main.go")
	if err != nil {
		t.Fatalf("FindFactsBySource: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("expected 2 facts for main.go, got %d", len(facts))
	}
}

func TestUpdateFactVerification(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.InsertFact("some claim", "file.go", 0)

	if err := s.UpdateFactVerification(id); err != nil {
		t.Fatalf("UpdateFactVerification: %v", err)
	}

	facts, _ := s.FindFacts("some claim", 10)
	if len(facts) == 0 {
		t.Fatal("fact disappeared after update")
	}
}

func TestDeleteFact(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.InsertFact("deletable claim", "file.go", 0)

	if err := s.DeleteFact(id); err != nil {
		t.Fatalf("DeleteFact: %v", err)
	}

	facts, _ := s.FindFacts("deletable claim", 10)
	if len(facts) != 0 {
		t.Errorf("expected fact to be deleted, got %d results", len(facts))
	}
}

func TestFindFacts_NoMatches(t *testing.T) {
	s := newTestStore(t)
	facts, err := s.FindFacts("this query matches nothing", 10)
	if err != nil {
		t.Fatalf("FindFacts: %v", err)
	}
	if facts == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(facts) != 0 {
		t.Errorf("expected 0 results, got %d", len(facts))
	}
}

// newTestStore creates a Store backed by a temp database for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
