package ctxwin

import (
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

// helper: build a tool call ID and its paired result message
func toolPair(callID string) (core.Message, core.Message) {
	call := core.NewAssistantMessageWithTools("", []core.ToolCall{
		{ID: callID, Type: "function", Function: core.FunctionCall{Name: "read_file", Arguments: "{}"}},
	})
	result := core.NewToolMessage(callID, "file contents")
	return call, result
}

func TestFindCompactionBoundary_NoPairs(t *testing.T) {
	msgs := []core.Message{
		core.NewSystemMessage("system"),
		core.NewUserMessage("hello"),
		core.NewAssistantMessage("response 1"),
		core.NewUserMessage("continue"),
		core.NewAssistantMessage("response 2"),
	}
	boundary := FindCompactionBoundary(msgs, 2)
	// Should keep last 2: boundary = 5 - 2 = 3
	if boundary != 3 {
		t.Errorf("expected boundary=3, got %d", boundary)
	}
}

func TestFindCompactionBoundary_PairAtBoundary(t *testing.T) {
	call, result := toolPair("call_1")
	msgs := []core.Message{
		core.NewUserMessage("old 1"),
		core.NewUserMessage("old 2"),
		call,   // index 2
		result, // index 3
		core.NewAssistantMessage("final response"),
	}
	// targetKeepCount=3 → proposed boundary = 5-3 = 2, which is the tool call
	// The result is at 3, so boundary must move back to keep the pair.
	boundary := FindCompactionBoundary(msgs, 3)
	if boundary > 2 {
		t.Errorf("expected boundary <= 2 to preserve tool pair, got %d", boundary)
	}
}

func TestFindCompactionBoundary_LargeKeepCount(t *testing.T) {
	msgs := []core.Message{
		core.NewUserMessage("a"),
		core.NewUserMessage("b"),
	}
	boundary := FindCompactionBoundary(msgs, 100)
	if boundary != 0 {
		t.Errorf("expected boundary=0 when targetKeepCount >= len, got %d", boundary)
	}
}

func TestValidateToolPairs_Valid(t *testing.T) {
	call, result := toolPair("call_1")
	msgs := []core.Message{
		core.NewUserMessage("hello"),
		call,
		result,
	}
	if err := ValidateToolPairs(msgs); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateToolPairs_OrphanedResult(t *testing.T) {
	result := core.NewToolMessage("call_missing", "some output")
	msgs := []core.Message{
		core.NewUserMessage("hello"),
		result, // no matching assistant tool call
	}
	if err := ValidateToolPairs(msgs); err == nil {
		t.Error("expected error for orphaned tool result, got nil")
	}
}

func TestValidateToolPairs_NoTools(t *testing.T) {
	msgs := []core.Message{
		core.NewSystemMessage("system"),
		core.NewUserMessage("hello"),
		core.NewAssistantMessage("response"),
	}
	if err := ValidateToolPairs(msgs); err != nil {
		t.Errorf("expected no error for messages with no tools, got: %v", err)
	}
}
