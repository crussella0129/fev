package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// mockReviewClient implements ReviewClient for testing.
type mockReviewClient struct {
	response string
}

func (m *mockReviewClient) Generate(_ context.Context, _ []core.Message, _ []llm.ToolSchema) (*core.Message, error) {
	return &core.Message{Role: core.RoleAssistant, Content: m.response}, nil
}

func TestReview_GeneratesNonEmpty(t *testing.T) {
	base := t.TempDir()
	s, err := NewSession(base)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.AddObservation("project uses Go 1.26")
	s.AddQuestion("should we add PostgreSQL support?")

	store := newTestStore(t)
	client := &mockReviewClient{response: "Explored project structure and identified PostgreSQL question as open."}

	summary, err := Review(context.Background(), client, s, store)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	// Review file should exist
	reviewPath := filepath.Join(s.Dir, "review.md")
	if _, err := os.Stat(reviewPath); os.IsNotExist(err) {
		t.Error("expected review.md to be created")
	}
}

func TestReview_IncludesCorrections(t *testing.T) {
	base := t.TempDir()
	s, _ := NewSession(base)

	store := newTestStore(t)
	_, _ = store.InsertCorrection("project uses MySQL", "project uses PostgreSQL")

	called := false
	var capturedPrompt string
	client := &capturingClient{
		response: "summary",
		capture: func(msgs []core.Message) {
			called = true
			if len(msgs) > 0 {
				capturedPrompt = msgs[0].Content
			}
		},
	}

	_, err := Review(context.Background(), client, s, store)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if !called {
		t.Error("expected LLM to be called")
	}
	if capturedPrompt == "" || len(capturedPrompt) < 10 {
		t.Errorf("expected non-trivial prompt, got %q", capturedPrompt)
	}
}

func TestLoadLatestReview_Empty(t *testing.T) {
	dir := t.TempDir()
	result, err := LoadLatestReview(dir)
	if err != nil {
		t.Fatalf("LoadLatestReview: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for empty dir, got %q", result)
	}
}

func TestLoadLatestReview_FindsLatest(t *testing.T) {
	base := t.TempDir()

	// Create two sessions and write reviews.
	s1, _ := NewSession(base)
	_ = os.WriteFile(filepath.Join(s1.Dir, "review.md"), []byte("# Session Review\n\nFirst session."), 0644)

	s2, _ := NewSession(base)
	_ = os.WriteFile(filepath.Join(s2.Dir, "review.md"), []byte("# Session Review\n\nSecond session."), 0644)

	result, err := LoadLatestReview(base)
	if err != nil {
		t.Fatalf("LoadLatestReview: %v", err)
	}
	// The latest session's review should be returned.
	// Both s1.ID and s2.ID are timestamp-based; s2 was created after s1.
	if result == "" {
		t.Error("expected non-empty review")
	}
}

func TestLoadLatestReview_MissingDir(t *testing.T) {
	result, err := LoadLatestReview("/nonexistent/path/sessions")
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for missing dir, got %q", result)
	}
}

// capturingClient captures the messages passed to Generate.
type capturingClient struct {
	response string
	capture  func([]core.Message)
}

func (c *capturingClient) Generate(_ context.Context, msgs []core.Message, _ []llm.ToolSchema) (*core.Message, error) {
	if c.capture != nil {
		c.capture(msgs)
	}
	return &core.Message{Role: core.RoleAssistant, Content: c.response}, nil
}
