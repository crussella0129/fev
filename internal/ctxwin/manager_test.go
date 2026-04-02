package ctxwin

import (
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestManager_UsageTracking(t *testing.T) {
	m := NewManager(8192, 2048)
	msgs := []core.Message{
		core.NewSystemMessage("you are helpful"),
		core.NewUserMessage("hello"),
	}
	usage := m.EstimateUsage(msgs)
	if usage <= 0 {
		t.Errorf("expected positive usage, got %d", usage)
	}
	if usage >= 8192 {
		t.Errorf("expected usage < max, got %d", usage)
	}
}

func TestManager_NeedsCompaction(t *testing.T) {
	m := NewManager(100, 20) // small window for testing
	// Create messages that would exceed 75% of 100 tokens
	var msgs []core.Message
	for i := 0; i < 50; i++ {
		msgs = append(msgs, core.NewUserMessage("this is a test message with enough words to use tokens"))
	}
	if !m.NeedsCompaction(msgs) {
		t.Error("expected NeedsCompaction=true for large message set")
	}
}

func TestManager_NeedsCompaction_Small(t *testing.T) {
	m := NewManager(8192, 2048)
	msgs := []core.Message{core.NewUserMessage("hi")}
	if m.NeedsCompaction(msgs) {
		t.Error("expected NeedsCompaction=false for small message set")
	}
}

func TestManager_Trim(t *testing.T) {
	// Budget = 30 - 6 = 24 tokens available after reserve.
	// System message "system" costs ~6 tokens, leaving 18 for non-system.
	// Each non-system message costs ~8 tokens, so only 2 can fit.
	m := NewManager(30, 6)
	msgs := []core.Message{
		core.NewSystemMessage("system"),
		core.NewUserMessage("old message 1"),
		core.NewAssistantMessage("old response 1"),
		core.NewUserMessage("old message 2"),
		core.NewAssistantMessage("old response 2"),
		core.NewUserMessage("recent message"),
		core.NewAssistantMessage("recent response"),
	}
	trimmed := m.Trim(msgs)
	// System message must be first
	if trimmed[0].Role != core.RoleSystem {
		t.Error("expected system message preserved as first")
	}
	// Should have fewer messages
	if len(trimmed) >= len(msgs) {
		t.Errorf("expected trimmed to be shorter, got %d (original %d)", len(trimmed), len(msgs))
	}
	// Last message should be preserved
	if trimmed[len(trimmed)-1].Content != "recent response" {
		t.Errorf("expected most recent message preserved, got %q", trimmed[len(trimmed)-1].Content)
	}
}

func TestManager_Available(t *testing.T) {
	m := NewManager(8192, 2048)
	msgs := []core.Message{core.NewUserMessage("hello")}
	avail := m.Available(msgs)
	if avail <= 0 {
		t.Errorf("expected positive available, got %d", avail)
	}
	if avail >= 8192 {
		t.Errorf("expected available < max, got %d", avail)
	}
}
