// internal/tools/executor_test.go
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type slowTool struct{ mockTool }

func (s *slowTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	select {
	case <-time.After(5 * time.Second):
		return &ToolResult{Output: "done"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	tool := &mockTool{name: "fast", result: "ok"}
	result, err := ExecuteWithTimeout(context.Background(), tool, json.RawMessage(`{}`), 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "ok" {
		t.Errorf("expected 'ok', got %q", result.Output)
	}
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	tool := &slowTool{mockTool{name: "slow"}}
	_, err := ExecuteWithTimeout(context.Background(), tool, json.RawMessage(`{}`), 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestTruncateOutput(t *testing.T) {
	long := strings.Repeat("a", 40000) // ~10K tokens at 4 chars/token
	result := &ToolResult{Output: long}
	truncated := MaybeTruncate(result, 8000)
	if !truncated.Truncated {
		t.Error("expected truncated=true for large output")
	}
	if len(truncated.Output) >= len(long) {
		t.Error("expected output to be shorter after truncation")
	}
}
