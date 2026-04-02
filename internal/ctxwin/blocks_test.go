package ctxwin

import "testing"

func TestBlockType_String(t *testing.T) {
	tests := []struct {
		bt   BlockType
		want string
	}{
		{BlockStatic, "static"},
		{BlockActive, "active"},
		{BlockArchive, "archive"},
		{BlockType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.bt.String(); got != tt.want {
			t.Errorf("BlockType(%d).String() = %q, want %q", tt.bt, got, tt.want)
		}
	}
}
