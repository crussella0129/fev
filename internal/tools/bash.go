// internal/tools/bash.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

const defaultBashTimeout = 120 // seconds
const maxBashTimeout = 600     // seconds

type bashArgs struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Timeout     int    `json:"timeout,omitempty"` // seconds
}

// BashTool executes shell commands with safety restrictions.
type BashTool struct {
	workspace *core.Workspace
}

func NewBashTool(ws *core.Workspace) *BashTool {
	return &BashTool{workspace: ws}
}

func (t *BashTool) Name() string { return "bash" }
func (t *BashTool) Description() string {
	return "Execute a shell command. Requires a description of what the command does. Dangerous commands and shell injection are blocked."
}
func (t *BashTool) PermissionTier() permission.PermissionTier { return permission.Dangerous }

func (t *BashTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "The command to execute"},
			"description": {"type": "string", "description": "Clear description of what this command does"},
			"timeout": {"type": "integer", "description": "Timeout in seconds (default 120, max 600)"}
		},
		"required": ["command", "description"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args bashArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Description == "" {
		return nil, fmt.Errorf("description is required for bash commands")
	}

	if permission.IsCommandDangerous(args.Command) {
		return nil, fmt.Errorf("command blocked: %q is dangerous", args.Command)
	}

	if permission.IsInjectionAttempt(args.Command) {
		return nil, fmt.Errorf("command blocked: detected shell injection in %q", args.Command)
	}

	timeout := args.Timeout
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}
	if timeout > maxBashTimeout {
		timeout = maxBashTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Split command into program + args (list-based, no shell interpretation)
	parts := splitCommand(args.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = t.workspace.CWD()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	exitCode := 0
	if err != nil {
		// Check context cancellation first — a killed process also returns ExitError
		if ctx.Err() != nil {
			return nil, fmt.Errorf("command timed out after %v", duration)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return &ToolResult{
		Output:   output,
		ExitCode: exitCode,
	}, nil
}

// splitCommand splits a command string into program and arguments.
// On Windows, uses cmd /c; on Unix, uses sh -c.
func splitCommand(cmd string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", cmd}
	}
	return []string{"sh", "-c", cmd}
}
