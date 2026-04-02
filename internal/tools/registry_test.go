// internal/tools/registry_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/crussella0129/fev/internal/permission"
)

// mockTool is a simple tool for testing.
type mockTool struct {
	name   string
	result string
}

func (m *mockTool) Name() string                                    { return m.name }
func (m *mockTool) Description() string                             { return "mock tool" }
func (m *mockTool) Schema() json.RawMessage                         { return json.RawMessage(`{"type":"object"}`) }
func (m *mockTool) PermissionTier() permission.PermissionTier       { return permission.ReadOnly }

func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Output: m.result}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tool := &mockTool{name: "test_tool", result: "ok"}

	reg.Register(tool)

	got := reg.Get("test_tool")
	if got == nil {
		t.Fatal("expected to find registered tool")
	}
	if got.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", got.Name())
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "b_tool"})
	reg.Register(&mockTool{name: "a_tool"})

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(list))
	}
	// Should be sorted
	if list[0] != "a_tool" || list[1] != "b_tool" {
		t.Errorf("expected sorted list [a_tool, b_tool], got %v", list)
	}
}

func TestRegistry_ToOpenAISchemas(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "read_file"})

	schemas := reg.ToOpenAISchemas()
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	if schemas[0].Function.Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", schemas[0].Function.Name)
	}
}
