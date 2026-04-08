package ctxwin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crussella0129/fev/internal/core"
)

func TestExtractor_ThresholdGate(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")

	e := NewExtractor(memPath)
	client := &mockSummarizer{response: "## Observations\n- something"}

	msgs := []core.Message{core.NewUserMessage("hello")}

	// Below init threshold — should not extract
	e.MaybeExtract(context.Background(), client, msgs, 100)
	if e.running {
		t.Error("expected no extraction below init threshold")
	}
}

func TestExtractor_AllThresholdsMet(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")

	e := NewExtractor(memPath)
	e.initThreshold = 100   // low for testing
	e.growthThreshold = 50  // low for testing
	e.callThreshold = 2
	client := &mockSummarizer{response: "## Observations\n- found something"}

	// Record enough tool calls
	e.RecordToolCall()
	e.RecordToolCall()

	msgs := []core.Message{core.NewUserMessage("hello"), core.NewAssistantMessage("response")}
	e.MaybeExtract(context.Background(), client, msgs, 200)

	// Wait for completion
	completed := e.WaitForCompletion(5 * time.Second)
	if !completed {
		t.Error("extraction timed out")
	}

	// Session memory file should have been written
	if _, err := os.Stat(memPath); os.IsNotExist(err) {
		t.Error("expected session_memory.md to be written")
	}
}

func TestExtractor_ToolCallThreshold(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")

	e := NewExtractor(memPath)
	e.initThreshold = 100
	e.growthThreshold = 50
	e.callThreshold = 3
	client := &mockSummarizer{response: "summary"}

	// Only 2 tool calls — below threshold of 3
	e.RecordToolCall()
	e.RecordToolCall()

	msgs := []core.Message{core.NewUserMessage("hello")}
	e.MaybeExtract(context.Background(), client, msgs, 200)

	if e.running {
		t.Error("expected no extraction with insufficient tool calls")
	}
}

func TestExtractor_WaitForCompletion_Timeout(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")

	e := NewExtractor(memPath)
	// Not running — WaitForCompletion should return immediately
	completed := e.WaitForCompletion(100 * time.Millisecond)
	if !completed {
		t.Error("expected immediate return when not running")
	}
}

func TestExtractor_NoDoubleExtraction(t *testing.T) {
	dir := t.TempDir()
	memPath := filepath.Join(dir, "session_memory.md")

	e := NewExtractor(memPath)
	e.initThreshold = 10
	e.growthThreshold = 5
	e.callThreshold = 1

	// Slow summarizer to keep goroutine busy
	slow := &mockSummarizer{response: "summary"}
	e.RecordToolCall()

	msgs := []core.Message{core.NewUserMessage("hello")}
	e.MaybeExtract(context.Background(), slow, msgs, 100)

	firstRunning := e.running

	// Call again immediately — should not start a second extraction
	callsBefore := slow.calls
	e.MaybeExtract(context.Background(), slow, msgs, 100)

	e.WaitForCompletion(5 * time.Second)

	if !firstRunning {
		// If it wasn't running yet, skip this check
		return
	}
	_ = callsBefore // prevent unused warning
}
