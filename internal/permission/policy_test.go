package permission

import (
	"testing"
)

func TestPermissionTier_String(t *testing.T) {
	if ReadOnly.String() != "read-only" {
		t.Errorf("expected 'read-only', got %q", ReadOnly.String())
	}
	if Write.String() != "write" {
		t.Errorf("expected 'write', got %q", Write.String())
	}
	if Dangerous.String() != "dangerous" {
		t.Errorf("expected 'dangerous', got %q", Dangerous.String())
	}
}

func TestRequiresConfirmation(t *testing.T) {
	if RequiresConfirmation(ReadOnly, true) {
		t.Error("read-only should never require confirmation")
	}
	if RequiresConfirmation(Write, true) {
		t.Error("write with confirm=true should not require confirmation (only dangerous does)")
	}
	if !RequiresConfirmation(Dangerous, true) {
		t.Error("dangerous with confirm=true should require confirmation")
	}
	if RequiresConfirmation(Dangerous, false) {
		t.Error("dangerous with confirm=false should not require confirmation")
	}
}
