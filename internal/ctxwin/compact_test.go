package ctxwin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// mockSummarizer is a controllable Summarizer for testing.
type mockSummarizer struct {
	response string
	err      error
	calls    int
}

func (m *mockSummarizer) Generate(_ context.Context, _ []core.Message, _ []llm.ToolSchema) (*core.Message, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &core.Message{Role: core.RoleAssistant, Content: m.response}, nil
}

func buildConversation(n int) []core.Message {
	msgs := []core.Message{core.NewSystemMessage("system")}
	for i := 0; i < n; i++ {
		msgs = append(msgs, core.NewUserMessage("message"))
		msgs = append(msgs, core.NewAssistantMessage("response"))
	}
	return msgs
}

func TestCompact_UsesSessionMemory(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")
	_ = os.WriteFile(memPath, []byte("## Observations\n- project uses Go"), 0644)

	c := NewCompactor()
	client := &mockSummarizer{response: "LLM summary"}
	msgs := buildConversation(10)

	result, err := c.Compact(context.Background(), msgs, client, memPath)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if len(result) >= len(msgs) {
		t.Errorf("expected compacted messages to be shorter, got %d >= %d", len(result), len(msgs))
	}
	// LLM should NOT have been called since session_memory.md exists
	if client.calls != 0 {
		t.Errorf("expected 0 LLM calls (session memory used), got %d", client.calls)
	}
}

func TestCompact_FallsBackToLLM(t *testing.T) {
	c := NewCompactor()
	client := &mockSummarizer{response: "LLM-generated summary"}
	msgs := buildConversation(10)

	result, err := c.Compact(context.Background(), msgs, client, "")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if len(result) >= len(msgs) {
		t.Errorf("expected compacted messages to be shorter, got %d >= %d", len(result), len(msgs))
	}
	if client.calls == 0 {
		t.Error("expected LLM to be called when no session memory file")
	}
}

func TestCompact_CircuitBreaker(t *testing.T) {
	c := NewCompactor()
	client := &mockSummarizer{err: errors.New("LLM unavailable")}
	msgs := buildConversation(10)

	// Trip the circuit breaker with 3 failures.
	for i := 0; i < 3; i++ {
		_, _ = c.Compact(context.Background(), msgs, client, "")
	}
	if !c.CircuitBreakerTripped() {
		t.Error("expected circuit breaker to trip after 3 failures")
	}

	// 4th call should use truncation (no LLM call).
	prevCalls := client.calls
	result, err := c.Compact(context.Background(), msgs, client, "")
	if err != nil {
		t.Fatalf("Compact after circuit trip: %v", err)
	}
	if client.calls != prevCalls {
		t.Errorf("expected no LLM call after circuit trip, got %d more calls", client.calls-prevCalls)
	}
	if len(result) >= len(msgs) {
		t.Errorf("expected truncation to reduce messages, got %d >= %d", len(result), len(msgs))
	}
}

func TestCompact_CircuitBreakerResets(t *testing.T) {
	c := NewCompactor()
	failClient := &mockSummarizer{err: errors.New("fail")}
	successClient := &mockSummarizer{response: "summary"}
	msgs := buildConversation(10)

	// Trip breaker
	for i := 0; i < 3; i++ {
		_, _ = c.Compact(context.Background(), msgs, failClient, "")
	}

	// Reset by using a successful client — but breaker is tripped so it truncates.
	// We reset manually to test circuit breaker reset logic.
	c.circuitBreaker = 0
	_, err := c.Compact(context.Background(), msgs, successClient, "")
	if err != nil {
		t.Fatalf("Compact after manual reset: %v", err)
	}
	if c.CircuitBreakerTripped() {
		t.Error("expected circuit breaker to reset after successful compaction")
	}
}

func TestCompact_TruncationFallback(t *testing.T) {
	c := NewCompactor()
	c.circuitBreaker = c.maxFailures // pre-trip
	client := &mockSummarizer{}
	msgs := buildConversation(10)

	result, err := c.Compact(context.Background(), msgs, client, "")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	// Should have fewer messages than original
	if len(result) >= len(msgs) {
		t.Errorf("expected truncation to reduce messages, got %d >= %d", len(result), len(msgs))
	}
	// System message must be preserved
	if result[0].Role != core.RoleSystem {
		t.Error("expected system message preserved after truncation")
	}
}
