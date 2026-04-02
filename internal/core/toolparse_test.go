package core

import (
	"testing"
)

func TestDeduplicateToolCalls_NoDupes(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		{ID: "2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"b.go"}`}},
	}
	deduped := DeduplicateToolCalls(calls)
	if len(deduped) != 2 {
		t.Errorf("expected 2 calls, got %d", len(deduped))
	}
}

func TestDeduplicateToolCalls_WithDupes(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		{ID: "2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
	}
	deduped := DeduplicateToolCalls(calls)
	if len(deduped) != 1 {
		t.Errorf("expected 1 call after dedup, got %d", len(deduped))
	}
}

func TestToolCallHash_IgnoresID(t *testing.T) {
	a := ToolCall{ID: "1", Function: FunctionCall{Name: "bash", Arguments: `{"command":"ls"}`}}
	b := ToolCall{ID: "2", Function: FunctionCall{Name: "bash", Arguments: `{"command":"ls"}`}}

	if ToolCallHash(a) != ToolCallHash(b) {
		t.Error("same name+args should produce same hash regardless of ID")
	}
}

func TestToolCallHash_DifferentArgs(t *testing.T) {
	a := ToolCall{ID: "1", Function: FunctionCall{Name: "bash", Arguments: `{"command":"ls"}`}}
	c := ToolCall{ID: "3", Function: FunctionCall{Name: "bash", Arguments: `{"command":"pwd"}`}}

	if ToolCallHash(a) == ToolCallHash(c) {
		t.Error("different args should produce different hash")
	}
}
