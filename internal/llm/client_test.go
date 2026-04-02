package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestGenerate_SimpleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := ChatResponse{
			ID: "test-1",
			Choices: []Choice{
				{
					Message:      core.Message{Role: core.RoleAssistant, Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	messages := []core.Message{core.NewUserMessage("hi")}

	resp, err := client.Generate(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", resp.Content)
	}
}

func TestGenerate_ToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			ID: "test-2",
			Choices: []Choice{
				{
					Message: core.Message{
						Role: core.RoleAssistant,
						ToolCalls: []core.ToolCall{
							{
								ID:   "call_abc",
								Type: "function",
								Function: core.FunctionCall{
									Name:      "read_file",
									Arguments: `{"path":"main.go"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: Usage{PromptTokens: 20, CompletionTokens: 15, TotalTokens: 35},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	messages := []core.Message{core.NewUserMessage("read main.go")}

	resp, err := client.Generate(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected read_file, got %q", resp.ToolCalls[0].Function.Name)
	}
}

func TestGenerate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal error"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	_, err := client.Generate(context.Background(), []core.Message{core.NewUserMessage("hi")}, nil)
	if err == nil {
		t.Fatal("expected error on 500 response")
	}
}
