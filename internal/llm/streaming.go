package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/crussella0129/fev/internal/core"
)

// StreamDelta is a single chunk from the SSE stream.
type StreamDelta struct {
	ID      string        `json:"id"`
	Choices []DeltaChoice `json:"choices"`
}

// DeltaChoice is a streaming choice.
type DeltaChoice struct {
	Delta        DeltaContent `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// DeltaContent holds incremental content or tool call fragments.
type DeltaContent struct {
	Content   string          `json:"content,omitempty"`
	ToolCalls []DeltaToolCall `json:"tool_calls,omitempty"`
}

// DeltaToolCall is an incremental tool call fragment.
type DeltaToolCall struct {
	Index    int               `json:"index"`
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function DeltaFunctionCall `json:"function,omitempty"`
}

// DeltaFunctionCall holds incremental function call data.
type DeltaFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// GenerateStream sends a streaming chat completion request.
// onChunk is called for each text content chunk.
// Returns the fully assembled message.
func (c *Client) GenerateStream(ctx context.Context, messages []core.Message, tools []ToolSchema, onChunk func(string)) (*core.Message, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	// Assemble the full message from streaming deltas
	var contentBuilder strings.Builder
	toolCallMap := make(map[int]*core.ToolCall) // index -> accumulated tool call

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		data, done := parseSSELine(line)
		if done {
			break
		}
		if data == "" {
			continue
		}

		var delta StreamDelta
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			continue // skip malformed chunks
		}

		for _, choice := range delta.Choices {
			// Accumulate text content
			if choice.Delta.Content != "" {
				contentBuilder.WriteString(choice.Delta.Content)
				if onChunk != nil {
					onChunk(choice.Delta.Content)
				}
			}

			// Accumulate tool calls
			for _, dtc := range choice.Delta.ToolCalls {
				tc, exists := toolCallMap[dtc.Index]
				if !exists {
					tc = &core.ToolCall{Type: "function"}
					toolCallMap[dtc.Index] = tc
				}
				if dtc.ID != "" {
					tc.ID = dtc.ID
				}
				if dtc.Function.Name != "" {
					tc.Function.Name = dtc.Function.Name
				}
				tc.Function.Arguments += dtc.Function.Arguments
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	// Build the final message
	msg := &core.Message{
		Role:    core.RoleAssistant,
		Content: contentBuilder.String(),
	}

	if len(toolCallMap) > 0 {
		msg.ToolCalls = make([]core.ToolCall, 0, len(toolCallMap))
		for i := 0; i < len(toolCallMap); i++ {
			if tc, ok := toolCallMap[i]; ok {
				msg.ToolCalls = append(msg.ToolCalls, *tc)
			}
		}
	}

	return msg, nil
}

// parseSSELine parses a single SSE line.
// Returns (data, isDone).
func parseSSELine(line string) (string, bool) {
	if strings.HasPrefix(line, "data: [DONE]") {
		return "", true
	}
	if strings.HasPrefix(line, "data: ") {
		return strings.TrimPrefix(line, "data: "), false
	}
	return "", false
}
