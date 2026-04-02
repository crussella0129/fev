package agent

import (
	"testing"
)

func TestRepeatThreshold(t *testing.T) {
	if repeatThreshold != 3 {
		t.Errorf("expected repeatThreshold=3, got %d", repeatThreshold)
	}
}
