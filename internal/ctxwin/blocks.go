// Package ctxwin manages context window token budgets and message trimming.
package ctxwin

// BlockType categorizes context content for compaction decisions.
type BlockType int

const (
	BlockStatic  BlockType = iota // System prompt, project config — never compacted
	BlockActive                   // Conversation, tool results — compactable
	BlockArchive                  // Compacted summaries — read-only reference
)

func (b BlockType) String() string {
	switch b {
	case BlockStatic:
		return "static"
	case BlockActive:
		return "active"
	case BlockArchive:
		return "archive"
	default:
		return "unknown"
	}
}
