package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/tools"
)

const repeatThreshold = 3

// loop runs the flat agent loop until the model emits a text-only response or max turns.
func (a *Agent) loop(ctx context.Context) (string, error) {
	maxTurns := a.config.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}

	repeatTracker := make(map[string]int) // hash → count

	for turn := 0; turn < maxTurns; turn++ {
		// Trim history if needed
		a.history = a.ctxMgr.Trim(a.history)

		// Generate
		toolSchemas := a.registry.ToOpenAISchemas()
		resp, err := a.client.Generate(ctx, a.history, toolSchemas)
		if err != nil {
			class := core.ClassifyError(err)
			if class == core.ErrorRetryable && turn < maxTurns-1 {
				backoff := time.Duration(1<<uint(turn)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			return "", fmt.Errorf("generation failed: %w", err)
		}

		// No tool calls → done
		if len(resp.ToolCalls) == 0 {
			a.history = append(a.history, core.NewAssistantMessage(resp.Content))
			return resp.Content, nil
		}

		// Deduplicate tool calls
		toolCalls := core.DeduplicateToolCalls(resp.ToolCalls)

		// Check repeat detection
		allRepeated := true
		for _, tc := range toolCalls {
			hash := core.ToolCallHash(tc)
			repeatTracker[hash]++
			if repeatTracker[hash] < repeatThreshold {
				allRepeated = false
			}
		}

		if allRepeated {
			// All tool calls have been seen 3+ times — break the loop
			text := resp.Content
			if text == "" {
				text = "[Agent stopped: repeated tool calls detected]"
			}
			a.history = append(a.history, core.NewAssistantMessage(text))
			return text, nil
		}

		// Add assistant message with tool calls
		a.history = append(a.history, core.NewAssistantMessageWithTools(resp.Content, toolCalls))

		// Execute each tool
		for _, tc := range toolCalls {
			tool := a.registry.Get(tc.Function.Name)
			if tool == nil {
				a.history = append(a.history, core.NewToolMessage(tc.ID, fmt.Sprintf("Error: unknown tool %q", tc.Function.Name)))
				continue
			}

			result, execErr := tools.ExecuteWithTimeout(ctx, tool, json.RawMessage(tc.Function.Arguments), 120*time.Second)
			if execErr != nil {
				a.history = append(a.history, core.NewToolMessage(tc.ID, "Error: "+execErr.Error()))
				continue
			}

			// Truncate large output
			result = tools.MaybeTruncate(result, 8000)
			a.history = append(a.history, core.NewToolMessage(tc.ID, result.Output))
		}
	}

	return "", fmt.Errorf("max turns (%d) exceeded", maxTurns)
}
