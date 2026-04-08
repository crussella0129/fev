package ctxwin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// Summarizer is any LLM client that can generate a completion.
// Structurally compatible with agent.LLMClient — no import cycle needed.
type Summarizer interface {
	Generate(ctx context.Context, messages []core.Message, tools []llm.ToolSchema) (*core.Message, error)
}

// Compactor compresses message history when the context window fills up.
// It uses an LLM to summarize old messages, with a circuit breaker that
// falls back to simple truncation after 3 consecutive LLM failures.
type Compactor struct {
	circuitBreaker int
	maxFailures    int
}

// NewCompactor creates a Compactor with a 3-strike circuit breaker.
func NewCompactor() *Compactor {
	return &Compactor{maxFailures: 3}
}

// Compact reduces message history to fit within the context window.
//
// If the circuit breaker has tripped (3 consecutive failures), it falls back
// to simple truncation. Otherwise it attempts LLM-based summarization:
//  1. Read sessionMemoryPath as the summary (if it exists).
//  2. Otherwise ask the LLM to summarize the oldest 50% of messages.
//  3. Replace old messages with the summary using FindCompactionBoundary.
func (c *Compactor) Compact(
	ctx context.Context,
	messages []core.Message,
	client Summarizer,
	sessionMemoryPath string,
) ([]core.Message, error) {
	if c.circuitBreaker >= c.maxFailures {
		return c.truncate(messages), nil
	}

	compacted, err := c.summarize(ctx, messages, client, sessionMemoryPath)
	if err != nil {
		c.circuitBreaker++
		return c.truncate(messages), nil
	}

	c.circuitBreaker = 0
	return compacted, nil
}

// CircuitBreakerTripped returns true if compaction has failed 3+ times in a row.
func (c *Compactor) CircuitBreakerTripped() bool {
	return c.circuitBreaker >= c.maxFailures
}

// summarize attempts LLM-based compaction.
func (c *Compactor) summarize(
	ctx context.Context,
	messages []core.Message,
	client Summarizer,
	sessionMemoryPath string,
) ([]core.Message, error) {
	// Separate system message from conversation.
	var system []core.Message
	var conversation []core.Message
	for _, m := range messages {
		if m.Role == core.RoleSystem {
			system = append(system, m)
		} else {
			conversation = append(conversation, m)
		}
	}

	if len(conversation) < 4 {
		return messages, fmt.Errorf("not enough messages to compact")
	}

	// Find the safe boundary — keep the most recent 50% of conversation.
	keepCount := len(conversation) / 2
	boundary := FindCompactionBoundary(conversation, keepCount)
	toSummarize := conversation[:boundary]
	toKeep := conversation[boundary:]

	// Get summary text.
	summary, err := c.getSummary(ctx, client, sessionMemoryPath, toSummarize)
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}

	summaryMsg := core.NewSystemMessage("[Compacted summary of earlier conversation]\n" + summary)
	result := append(system, summaryMsg)
	result = append(result, toKeep...)
	return result, nil
}

// getSummary returns a summary either from the session memory file or from the LLM.
func (c *Compactor) getSummary(
	ctx context.Context,
	client Summarizer,
	sessionMemoryPath string,
	toSummarize []core.Message,
) (string, error) {
	// Prefer the session memory file if it exists.
	if sessionMemoryPath != "" {
		if data, err := os.ReadFile(sessionMemoryPath); err == nil && len(data) > 0 {
			return string(data), nil
		}
	}

	// Fall back to asking the LLM.
	return c.llmSummarize(ctx, client, toSummarize)
}

// llmSummarize asks the LLM to summarize a slice of messages.
func (c *Compactor) llmSummarize(
	ctx context.Context,
	client Summarizer,
	messages []core.Message,
) (string, error) {
	var b strings.Builder
	for _, m := range messages {
		fmt.Fprintf(&b, "[%s]: %s\n", m.Role, m.Content)
	}

	prompt := "Summarize the following conversation history concisely. " +
		"Focus on decisions made, files read, and open questions. " +
		"Be brief — this summary replaces the original messages in the context window.\n\n" +
		b.String()

	result, err := client.Generate(ctx, []core.Message{
		core.NewUserMessage(prompt),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("LLM summarize: %w", err)
	}
	return result.Content, nil
}

// truncate removes the oldest non-system messages as a last resort.
func (c *Compactor) truncate(messages []core.Message) []core.Message {
	var system []core.Message
	var rest []core.Message
	for _, m := range messages {
		if m.Role == core.RoleSystem {
			system = append(system, m)
		} else {
			rest = append(rest, m)
		}
	}

	// Drop oldest 50% of non-system messages.
	if len(rest) > 1 {
		rest = rest[len(rest)/2:]
	}

	return append(system, rest...)
}
