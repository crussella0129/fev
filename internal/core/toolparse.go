package core

import (
	"crypto/sha256"
	"fmt"
)

// ToolCallHash produces a hash of a tool call based on name + arguments (ignoring ID).
// Used for deduplication and repeat detection.
func ToolCallHash(tc ToolCall) string {
	h := sha256.New()
	h.Write([]byte(tc.Function.Name))
	h.Write([]byte(tc.Function.Arguments))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// DeduplicateToolCalls removes duplicate tool calls (same name + args, ignoring ID).
func DeduplicateToolCalls(calls []ToolCall) []ToolCall {
	if len(calls) <= 1 {
		return calls
	}
	seen := make(map[string]bool)
	var result []ToolCall
	for _, tc := range calls {
		hash := ToolCallHash(tc)
		if !seen[hash] {
			seen[hash] = true
			result = append(result, tc)
		}
	}
	return result
}
