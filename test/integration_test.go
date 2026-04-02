// test/integration_test.go
package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/agent"
	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/tools"
)

func TestIntegration_ReadFileViaAgent(t *testing.T) {
	// Setup workspace with a test file
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("Hello, Fev!"), 0644)
	ws, _ := core.NewWorkspace(dir)

	// Mock LLM server: first call returns tool call, second returns text
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp interface{}

		if callCount == 1 {
			// Model decides to read the file
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "read_file",
										"arguments": `{"path":"hello.txt"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
			}
		} else {
			// Model responds with text after seeing file contents
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "The file says: Hello, Fev!",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Wire everything
	client := llm.NewClient(server.URL+"/v1", "test-model")
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFileTool(ws))
	ctxMgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := agent.New(client, reg, ctxMgr, cfg)
	resp, err := a.Run(context.Background(), "What does hello.txt say?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(resp, "Hello, Fev!") {
		t.Errorf("expected response to contain file contents, got %q", resp)
	}
}

func TestIntegration_EditFileViaAgent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main\n\nfunc hello() {}\n"), 0644)
	ws, _ := core.NewWorkspace(dir)

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp interface{}

		if callCount == 1 {
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "edit_file",
										"arguments": `{"path":"code.go","old_string":"func hello() {}","new_string":"func hello() {\n\tfmt.Println(\"hello\")\n}"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]int{"total_tokens": 20},
			}
		} else {
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Done! Updated hello() to print a greeting.",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{"total_tokens": 15},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "test-model")
	reg := tools.NewRegistry()
	reg.Register(tools.NewEditFileTool(ws))
	ctxMgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := agent.New(client, reg, ctxMgr, cfg)
	_, err := a.Run(context.Background(), "Add a print statement to hello()")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify the file was actually edited
	content, _ := os.ReadFile(filepath.Join(dir, "code.go"))
	if !strings.Contains(string(content), "Println") {
		t.Errorf("expected edited file to contain Println, got %q", string(content))
	}
}
