package core

import (
	"encoding/json"
	"testing"
)

func TestNewSystemMessage(t *testing.T) {
	m := NewSystemMessage("you are helpful")
	if m.Role != RoleSystem {
		t.Errorf("expected role %q, got %q", RoleSystem, m.Role)
	}
	if m.Content != "you are helpful" {
		t.Errorf("expected content %q, got %q", "you are helpful", m.Content)
	}
}

func TestNewUserMessage(t *testing.T) {
	m := NewUserMessage("hello")
	if m.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, m.Role)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	m := NewAssistantMessage("hi there")
	if m.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, m.Role)
	}
}

func TestNewAssistantMessageWithTools(t *testing.T) {
	calls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "read_file",
				Arguments: `{"path": "/tmp/test.go"}`,
			},
		},
	}
	m := NewAssistantMessageWithTools("let me read that", calls)
	if m.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, m.Role)
	}
	if len(m.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(m.ToolCalls))
	}
	if m.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected tool name %q, got %q", "read_file", m.ToolCalls[0].Function.Name)
	}
}

func TestNewToolMessage(t *testing.T) {
	m := NewToolMessage("call_1", "file contents here")
	if m.Role != RoleTool {
		t.Errorf("expected role %q, got %q", RoleTool, m.Role)
	}
	if m.ToolCallID != "call_1" {
		t.Errorf("expected tool_call_id %q, got %q", "call_1", m.ToolCallID)
	}
}

func TestMessageJSON(t *testing.T) {
	m := NewAssistantMessageWithTools("", []ToolCall{
		{
			ID:   "call_abc",
			Type: "function",
			Function: FunctionCall{
				Name:      "bash",
				Arguments: `{"command": "ls"}`,
			},
		},
	})
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call after round-trip, got %d", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].ID != "call_abc" {
		t.Errorf("expected tool call ID %q, got %q", "call_abc", decoded.ToolCalls[0].ID)
	}
}
