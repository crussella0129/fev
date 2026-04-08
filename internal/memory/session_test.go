package memory

import (
	"os"
	"strings"
	"testing"
)

func TestNewSession_CreatesDirectory(t *testing.T) {
	base := t.TempDir()
	s, err := NewSession(base)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if _, err := os.Stat(s.Dir); os.IsNotExist(err) {
		t.Errorf("session dir not created: %s", s.Dir)
	}
	if _, err := os.Stat(s.Dir + "/tool-results"); os.IsNotExist(err) {
		t.Errorf("tool-results dir not created")
	}
	if s.ID == "" {
		t.Error("expected non-empty session ID")
	}
}

func TestSession_RoundTrip(t *testing.T) {
	base := t.TempDir()
	s, err := NewSession(base)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	s.AddObservation("The project uses Go 1.26")
	s.AddObservation("main.go is the entry point")
	s.AddDecision("Use SQLite for persistence")
	s.AddQuestion("Should we support PostgreSQL later?")
	s.SetPlan([]string{"implement store", "add facts table"})

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadSession(s.Dir)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	if len(loaded.Observations) != 2 {
		t.Errorf("expected 2 observations, got %d", len(loaded.Observations))
	}
	if loaded.Observations[0] != "The project uses Go 1.26" {
		t.Errorf("unexpected observation: %q", loaded.Observations[0])
	}
	if len(loaded.Decisions) != 1 {
		t.Errorf("expected 1 decision, got %d", len(loaded.Decisions))
	}
	if len(loaded.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(loaded.Questions))
	}
	if len(loaded.Plan) != 2 {
		t.Errorf("expected 2 plan items, got %d", len(loaded.Plan))
	}
}

func TestToolResultPath(t *testing.T) {
	base := t.TempDir()
	s, _ := NewSession(base)

	path := s.ToolResultPath("read_file")

	// Must be inside session dir
	if !strings.HasPrefix(path, s.Dir) {
		t.Errorf("ToolResultPath %q not inside session dir %q", path, s.Dir)
	}
	// Must contain tool name
	if !strings.Contains(path, "read_file") {
		t.Errorf("ToolResultPath %q doesn't contain tool name", path)
	}
}
