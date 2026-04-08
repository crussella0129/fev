package memory

import (
	"testing"
)

func TestInsertHunch_Find(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertHunch("project probably uses PostgreSQL", 0.6)
	if err != nil {
		t.Fatalf("InsertHunch: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	hunches, err := s.FindHunches("PostgreSQL", 10)
	if err != nil {
		t.Fatalf("FindHunches: %v", err)
	}
	if len(hunches) != 1 {
		t.Fatalf("expected 1 hunch, got %d", len(hunches))
	}
	if hunches[0].Confidence != 0.6 {
		t.Errorf("expected confidence 0.6, got %f", hunches[0].Confidence)
	}
}

func TestDeleteHunch(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.InsertHunch("deletable hunch", 0.5)

	if err := s.DeleteHunch(id); err != nil {
		t.Fatalf("DeleteHunch: %v", err)
	}

	hunches, _ := s.FindHunches("deletable hunch", 10)
	if len(hunches) != 0 {
		t.Errorf("expected hunch deleted, got %d", len(hunches))
	}
}

func TestInsertHunch_ConfidenceClamped(t *testing.T) {
	s := newTestStore(t)

	// Below 0 → clamped to 0
	idLow, _ := s.InsertHunch("low confidence hunch", -0.5)
	hunches, _ := s.FindHunches("low confidence hunch", 10)
	if len(hunches) > 0 && hunches[0].ID == idLow && hunches[0].Confidence != 0.0 {
		t.Errorf("expected confidence clamped to 0.0, got %f", hunches[0].Confidence)
	}

	// Above 1 → clamped to 1
	idHigh, _ := s.InsertHunch("high confidence hunch", 1.5)
	hunches, _ = s.FindHunches("high confidence hunch", 10)
	if len(hunches) > 0 && hunches[0].ID == idHigh && hunches[0].Confidence != 1.0 {
		t.Errorf("expected confidence clamped to 1.0, got %f", hunches[0].Confidence)
	}
}
