package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/permission"
	"github.com/crussella0129/fev/internal/tools"
)

// mockClient implements a simple LLM client for testing.
type mockClient struct {
	responses []*core.Message
	callIndex int
}

func (m *mockClient) Generate(ctx context.Context, messages []core.Message, toolSchemas []llm.ToolSchema) (*core.Message, error) {
	if m.callIndex >= len(m.responses) {
		return &core.Message{Role: core.RoleAssistant, Content: "done"}, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

type echoTool struct{}

func (e *echoTool) Name() string                                    { return "echo" }
func (e *echoTool) Description() string                             { return "echoes input" }
func (e *echoTool) Schema() json.RawMessage                         { return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`) }
func (e *echoTool) PermissionTier() permission.PermissionTier       { return permission.ReadOnly }
func (e *echoTool) Execute(ctx context.Context, args json.RawMessage) (*tools.ToolResult, error) {
	var a struct{ Text string `json:"text"` }
	json.Unmarshal(args, &a)
	return &tools.ToolResult{Output: "echo: " + a.Text}, nil
}

func TestAgent_SimpleResponse(t *testing.T) {
	client := &mockClient{
		responses: []*core.Message{
			{Role: core.RoleAssistant, Content: "Hello!"},
		},
	}

	reg := tools.NewRegistry()
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", resp)
	}
}

func TestAgent_ToolCallThenResponse(t *testing.T) {
	client := &mockClient{
		responses: []*core.Message{
			// First: model calls echo tool
			{
				Role: core.RoleAssistant,
				ToolCalls: []core.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: core.FunctionCall{
							Name:      "echo",
							Arguments: `{"text":"world"}`,
						},
					},
				},
			},
			// Second: model responds with text
			{Role: core.RoleAssistant, Content: "The echo said: world"},
		},
	}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "echo world")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp != "The echo said: world" {
		t.Errorf("expected 'The echo said: world', got %q", resp)
	}
}

func TestAgent_MaxTurns(t *testing.T) {
	// Model always calls a tool with unique arguments, so repeat detection
	// never triggers, and we hit the max turns limit instead.
	infiniteCalls := make([]*core.Message, 100)
	for i := range infiniteCalls {
		infiniteCalls[i] = &core.Message{
			Role: core.RoleAssistant,
			ToolCalls: []core.ToolCall{
				{ID: "call", Type: "function", Function: core.FunctionCall{
					Name:      "echo",
					Arguments: fmt.Sprintf(`{"text":"loop-%d"}`, i),
				}},
			},
		}
	}
	client := &mockClient{responses: infiniteCalls}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()
	cfg.Agent.MaxTurns = 3

	a := New(client, reg, mgr, cfg)
	_, err := a.Run(context.Background(), "loop forever")
	if err == nil {
		t.Fatal("expected max turns error")
	}
}

func TestAgent_RepeatDetection(t *testing.T) {
	// Same tool call 3 times → should break
	sameCall := &core.Message{
		Role: core.RoleAssistant,
		ToolCalls: []core.ToolCall{
			{ID: "call", Type: "function", Function: core.FunctionCall{Name: "echo", Arguments: `{"text":"same"}`}},
		},
	}
	client := &mockClient{
		responses: []*core.Message{sameCall, sameCall, sameCall, {Role: core.RoleAssistant, Content: "final"}},
	}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "test repeat")
	// Should not error — repeat detection should break the loop gracefully
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	_ = resp
}
