package ctxwin

import (
	"fmt"

	"github.com/crussella0129/fev/internal/core"
)

// FindCompactionBoundary returns the index at which messages should be split for
// compaction. Messages before the index are summarized; messages at and after
// the index are kept verbatim.
//
// The boundary guarantees that no tool-use / tool-result pairs are split:
// if the proposed boundary would separate an assistant tool-call message from
// its corresponding tool-result message(s), the boundary is moved earlier to
// include the entire pair.
func FindCompactionBoundary(messages []core.Message, targetKeepCount int) int {
	n := len(messages)
	if targetKeepCount >= n {
		return 0
	}

	boundary := n - targetKeepCount

	// Scan backward from the boundary: if the message at boundary is a tool
	// result, or the message just before boundary is a tool-use that has
	// results at or after boundary, move the boundary left.
	for boundary > 0 {
		// A tool result at the boundary means its paired call is before boundary.
		if messages[boundary].Role == core.RoleTool {
			boundary--
			continue
		}
		// An assistant message with tool calls whose results span the boundary.
		if messages[boundary-1].Role == core.RoleAssistant && len(messages[boundary-1].ToolCalls) > 0 {
			boundary--
			continue
		}
		break
	}

	return boundary
}

// ValidateToolPairs checks that every tool-result message has a matching
// preceding assistant tool-call message.
func ValidateToolPairs(messages []core.Message) error {
	// Build a set of all tool call IDs from assistant messages.
	callIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == core.RoleAssistant {
			for _, tc := range msg.ToolCalls {
				callIDs[tc.ID] = true
			}
		}
	}

	for i, msg := range messages {
		if msg.Role == core.RoleTool && msg.ToolCallID != "" {
			if !callIDs[msg.ToolCallID] {
				return fmt.Errorf("message[%d]: orphaned tool result for call ID %q (no matching tool call found)", i, msg.ToolCallID)
			}
		}
	}
	return nil
}
