package llm

import (
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text    string
		wantMin int
		wantMax int
	}{
		{"", 0, 0},
		{"hello", 1, 3},
		{"this is a test of the token estimation", 5, 15},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateTokens(%q) = %d, want [%d, %d]", tt.text, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	msgs := []core.Message{
		core.NewSystemMessage("you are helpful"),
		core.NewUserMessage("hello"),
		core.NewAssistantMessage("hi there"),
	}
	total := EstimateMessageTokens(msgs)
	if total < 5 {
		t.Errorf("expected at least 5 tokens, got %d", total)
	}
}
