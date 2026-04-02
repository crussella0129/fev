package llm

import (
	"github.com/crussella0129/fev/internal/core"
)

// EstimateTokens provides a rough token count using the 4-chars-per-token heuristic.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// EstimateMessageTokens estimates total tokens across a slice of messages.
// Includes a small overhead per message for role/formatting (~4 tokens each).
func EstimateMessageTokens(messages []core.Message) int {
	total := 0
	for _, m := range messages {
		total += 4 // role + formatting overhead
		total += EstimateTokens(m.Content)
		for _, tc := range m.ToolCalls {
			total += EstimateTokens(tc.Function.Name)
			total += EstimateTokens(tc.Function.Arguments)
			total += 4 // tool call overhead
		}
	}
	return total
}
