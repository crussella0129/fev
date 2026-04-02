// internal/tools/executor.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crussella0129/fev/internal/llm"
)

// ExecuteWithTimeout runs a tool with a context deadline.
func ExecuteWithTimeout(ctx context.Context, tool Tool, args json.RawMessage, timeout time.Duration) (*ToolResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan *ToolResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := tool.Execute(ctx, args)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool %q timed out after %v", tool.Name(), timeout)
	}
}

// MaybeTruncate truncates a tool result if it exceeds maxTokens.
// Returns the (possibly modified) result.
func MaybeTruncate(result *ToolResult, maxTokens int) *ToolResult {
	estimated := llm.EstimateTokens(result.Output)
	if estimated <= maxTokens {
		return result
	}

	// Keep approximately maxTokens worth of characters
	maxChars := maxTokens * 4
	if maxChars >= len(result.Output) {
		return result
	}

	return &ToolResult{
		Output:    result.Output[:maxChars] + "\n\n[truncated — output exceeded limit]",
		Truncated: true,
		FilePath:  result.FilePath,
		ExitCode:  result.ExitCode,
	}
}
