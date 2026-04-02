package permission

import (
	"testing"
)

func TestIsCommandDangerous(t *testing.T) {
	dangerous := []string{
		"rm -rf /",
		"sudo reboot",
		"mkfs.ext4 /dev/sda",
		"shutdown -h now",
		"dd if=/dev/zero of=/dev/sda",
	}
	for _, cmd := range dangerous {
		t.Run(cmd, func(t *testing.T) {
			if !IsCommandDangerous(cmd) {
				t.Errorf("expected %q to be dangerous", cmd)
			}
		})
	}
}

func TestIsCommandDangerous_Safe(t *testing.T) {
	safe := []string{"ls", "go test ./...", "cat main.go", "git status"}
	for _, cmd := range safe {
		t.Run(cmd, func(t *testing.T) {
			if IsCommandDangerous(cmd) {
				t.Errorf("expected %q to be safe", cmd)
			}
		})
	}
}

func TestIsInjectionAttempt(t *testing.T) {
	injections := []string{
		"ls; rm -rf /",
		"echo $(whoami)",
		"cat `hostname`",
		"ls && rm -rf /",
		"ls || true",
		"cat < /etc/passwd",
		"echo hello > /etc/passwd",
	}
	for _, cmd := range injections {
		t.Run(cmd, func(t *testing.T) {
			if !IsInjectionAttempt(cmd) {
				t.Errorf("expected %q to be an injection attempt", cmd)
			}
		})
	}
}

func TestIsInjectionAttempt_Clean(t *testing.T) {
	clean := []string{
		"go test ./...",
		"git log --oneline -10",
		"ls -la",
		"grep -r TODO .",
	}
	for _, cmd := range clean {
		t.Run(cmd, func(t *testing.T) {
			if IsInjectionAttempt(cmd) {
				t.Errorf("expected %q to be clean", cmd)
			}
		})
	}
}

func TestIsPathDangerous(t *testing.T) {
	dangerous := []string{
		"/etc/passwd",
		"/sys/kernel",
		"/proc/self",
		"/boot/vmlinuz",
	}
	for _, p := range dangerous {
		t.Run(p, func(t *testing.T) {
			if !IsPathDangerous(p) {
				t.Errorf("expected %q to be dangerous", p)
			}
		})
	}
}
