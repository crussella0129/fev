package ctxwin

import (
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// Manager tracks token budget and decides when to compact.
type Manager struct {
	MaxTokens     int
	ReserveTokens int
	CompactAt     float64 // fraction (default 0.75)
}

// NewManager creates a context window manager.
func NewManager(maxTokens, reserveTokens int) *Manager {
	return &Manager{
		MaxTokens:     maxTokens,
		ReserveTokens: reserveTokens,
		CompactAt:     0.75,
	}
}

// EstimateUsage returns the estimated token count for a message slice.
func (m *Manager) EstimateUsage(messages []core.Message) int {
	return llm.EstimateMessageTokens(messages)
}

// Available returns how many tokens remain for generation.
func (m *Manager) Available(messages []core.Message) int {
	used := m.EstimateUsage(messages)
	return m.MaxTokens - used - m.ReserveTokens
}

// NeedsCompaction returns true if usage exceeds the CompactAt threshold.
func (m *Manager) NeedsCompaction(messages []core.Message) bool {
	used := m.EstimateUsage(messages)
	threshold := float64(m.MaxTokens) * m.CompactAt
	return float64(used) > threshold
}

// Trim removes oldest non-system messages to fit within budget.
// Preserves: system messages (always first), most recent messages.
// This is the simple fallback — auto-compaction with summarization is in Plan 2.
func (m *Manager) Trim(messages []core.Message) []core.Message {
	budget := m.MaxTokens - m.ReserveTokens

	// Always keep the system message
	var system []core.Message
	var rest []core.Message
	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			system = append(system, msg)
		} else {
			rest = append(rest, msg)
		}
	}

	// Start from the end (most recent) and work backward
	result := make([]core.Message, 0, len(messages))
	result = append(result, system...)
	systemTokens := llm.EstimateMessageTokens(system)
	remaining := budget - systemTokens

	// Add messages from most recent to oldest until budget exhausted
	var kept []core.Message
	for i := len(rest) - 1; i >= 0; i-- {
		msgTokens := llm.EstimateMessageTokens([]core.Message{rest[i]})
		if remaining-msgTokens < 0 {
			break
		}
		remaining -= msgTokens
		kept = append(kept, rest[i])
	}

	// Reverse kept to restore chronological order
	for i := len(kept) - 1; i >= 0; i-- {
		result = append(result, kept[i])
	}

	return result
}
