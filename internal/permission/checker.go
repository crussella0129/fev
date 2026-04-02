// Package permission provides deny-lists, injection detection, and permission tiers.
package permission

import (
	"regexp"
	"strings"
)

var dangerousDirs = []string{
	"/etc", "/sys", "/proc", "/boot", "/dev", "/root",
	"C:\\Windows", "C:\\System32", "C:\\ProgramData",
}

var dangerousCommands = []string{
	"rm -rf /", "rm -rf /*", "mkfs", "dd if=", "shutdown", "reboot",
	"halt", "poweroff", "init 0", "init 6",
	":(){ :|:& };:", // fork bomb
}

var dangerousCommandPrefixes = []string{
	"sudo", "su ", "chmod 777", "chown root",
}

var injectionPattern = regexp.MustCompile("[;|&<>]|\\$\\(|`")

// IsCommandDangerous checks if a command matches the blocked commands list.
func IsCommandDangerous(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	for _, dc := range dangerousCommands {
		if strings.Contains(lower, dc) {
			return true
		}
	}
	for _, prefix := range dangerousCommandPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// IsInjectionAttempt checks for shell metacharacters that indicate injection.
func IsInjectionAttempt(cmd string) bool {
	return injectionPattern.MatchString(cmd)
}

// IsPathDangerous checks if a path targets a sensitive system directory.
func IsPathDangerous(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, dir := range dangerousDirs {
		normalizedDir := strings.ReplaceAll(dir, "\\", "/")
		if strings.HasPrefix(normalized, normalizedDir) {
			return true
		}
	}
	return false
}
