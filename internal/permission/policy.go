package permission

// PermissionTier categorizes tool safety levels.
type PermissionTier int

const (
	ReadOnly  PermissionTier = iota // Execute immediately
	Write                           // Show diff, configurable confirmation
	Dangerous                       // Require confirmation
)

// String returns a human-readable tier name.
func (p PermissionTier) String() string {
	switch p {
	case ReadOnly:
		return "read-only"
	case Write:
		return "write"
	case Dangerous:
		return "dangerous"
	default:
		return "unknown"
	}
}

// RequiresConfirmation returns true if the tier requires user approval.
// confirmDangerous is from config — if false, even dangerous tools run without asking.
func RequiresConfirmation(tier PermissionTier, confirmDangerous bool) bool {
	return tier == Dangerous && confirmDangerous
}
