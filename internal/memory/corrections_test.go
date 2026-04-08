package memory

import (
	"testing"
)

func TestInsertCorrection_RecentCorrections(t *testing.T) {
	s := newTestStore(t)

	_, err := s.InsertCorrection("project uses MySQL", "project uses PostgreSQL")
	if err != nil {
		t.Fatalf("InsertCorrection: %v", err)
	}

	corrections, err := s.RecentCorrections(10)
	if err != nil {
		t.Fatalf("RecentCorrections: %v", err)
	}
	if len(corrections) != 1 {
		t.Fatalf("expected 1 correction, got %d", len(corrections))
	}
	if corrections[0].WrongClaim != "project uses MySQL" {
		t.Errorf("wrong WrongClaim: %q", corrections[0].WrongClaim)
	}
	if corrections[0].RightClaim != "project uses PostgreSQL" {
		t.Errorf("wrong RightClaim: %q", corrections[0].RightClaim)
	}
}

func TestRecentCorrections_Order(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.InsertCorrection("first wrong", "first right")
	_, _ = s.InsertCorrection("second wrong", "second right")
	_, _ = s.InsertCorrection("third wrong", "third right")

	corrections, err := s.RecentCorrections(10)
	if err != nil {
		t.Fatalf("RecentCorrections: %v", err)
	}
	if len(corrections) != 3 {
		t.Fatalf("expected 3 corrections, got %d", len(corrections))
	}
	// Should be newest first
	if corrections[0].WrongClaim != "third wrong" {
		t.Errorf("expected newest first, got %q", corrections[0].WrongClaim)
	}
}

func TestRecentCorrections_Limit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		_, _ = s.InsertCorrection("wrong", "right")
	}

	corrections, err := s.RecentCorrections(3)
	if err != nil {
		t.Fatalf("RecentCorrections: %v", err)
	}
	if len(corrections) != 3 {
		t.Errorf("expected limit of 3, got %d", len(corrections))
	}
}
