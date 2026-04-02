// internal/tools/git.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type gitArgs struct {
	Operation string   `json:"operation"`
	Files     []string `json:"files,omitempty"`
	Message   string   `json:"message,omitempty"`
	Count     int      `json:"count,omitempty"`
}

// GitTool provides read and write git operations.
type GitTool struct {
	workspace *core.Workspace
}

func NewGitTool(ws *core.Workspace) *GitTool {
	return &GitTool{workspace: ws}
}

func (t *GitTool) Name() string { return "git" }
func (t *GitTool) Description() string {
	return "Git operations: status, diff, log, add, commit. Operations like push, pull, clone are not supported."
}

// PermissionTier returns Write. Read-only operations (status, diff, log) are safe
// but the tool also supports write operations (add, commit), so the tier must
// cover the most dangerous capability. Dynamic per-operation tiers would require
// splitting into separate tools.
func (t *GitTool) PermissionTier() permission.PermissionTier {
	return permission.Write
}

func (t *GitTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"operation": {"type": "string", "enum": ["status", "diff", "log", "add", "commit"], "description": "Git operation"},
			"files": {"type": "array", "items": {"type": "string"}, "description": "Files to add (for 'add' operation)"},
			"message": {"type": "string", "description": "Commit message (for 'commit' operation)"},
			"count": {"type": "integer", "description": "Number of log entries (for 'log' operation, default 10)"}
		},
		"required": ["operation"]
	}`)
}

func (t *GitTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args gitArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	switch args.Operation {
	case "status":
		return t.runGit(ctx, "status", "--short")
	case "diff":
		return t.runGit(ctx, "diff")
	case "log":
		count := args.Count
		if count <= 0 {
			count = 10
		}
		return t.runGit(ctx, "log", "--oneline", fmt.Sprintf("-%d", count))
	case "add":
		if len(args.Files) == 0 {
			return nil, fmt.Errorf("git add requires files")
		}
		// Validate each file path against workspace boundary
		for _, f := range args.Files {
			if _, err := t.workspace.Resolve(f); err != nil {
				return nil, fmt.Errorf("git add: %w", err)
			}
		}
		cmdArgs := append([]string{"add"}, args.Files...)
		return t.runGit(ctx, cmdArgs...)
	case "commit":
		if args.Message == "" {
			return nil, fmt.Errorf("git commit requires a message")
		}
		return t.runGit(ctx, "commit", "-m", args.Message)
	default:
		return nil, fmt.Errorf("unsupported git operation: %q (supported: status, diff, log, add, commit)", args.Operation)
	}
}

func (t *GitTool) runGit(ctx context.Context, args ...string) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = t.workspace.Root()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 && output == "" {
		output = stderr.String()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ToolResult{Output: output, ExitCode: exitErr.ExitCode()}, nil
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	if output == "" {
		output = "OK (no output)"
	}
	return &ToolResult{Output: output}, nil
}
