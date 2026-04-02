// internal/tools/edit_file.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type editFileArgs struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditFileTool performs surgical string replacement in files.
type EditFileTool struct {
	workspace *core.Workspace
}

func NewEditFileTool(ws *core.Workspace) *EditFileTool {
	return &EditFileTool{workspace: ws}
}

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "Replace a specific string in a file. old_string must be unique unless replace_all is true. Provide enough surrounding context to ensure uniqueness."
}
func (t *EditFileTool) PermissionTier() permission.PermissionTier { return permission.Write }

func (t *EditFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path"},
			"old_string": {"type": "string", "description": "Exact text to find and replace"},
			"new_string": {"type": "string", "description": "Replacement text"},
			"replace_all": {"type": "boolean", "description": "Replace all occurrences (default false)"}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}

func (t *EditFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args editFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.OldString == args.NewString {
		return nil, fmt.Errorf("old_string and new_string are identical")
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", args.Path, err)
	}

	original := string(content)
	count := strings.Count(original, args.OldString)

	if count == 0 {
		return nil, fmt.Errorf("old_string not found in %q", args.Path)
	}

	if !args.ReplaceAll && count > 1 {
		return nil, fmt.Errorf("old_string has %d occurrences in %q — not unique. Provide more surrounding context to make it unique, or set replace_all=true", count, args.Path)
	}

	var newContent string
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(original, args.OldString, args.NewString)
	} else {
		newContent = strings.Replace(original, args.OldString, args.NewString, 1)
	}

	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	replacements := count
	if !args.ReplaceAll {
		replacements = 1
	}

	return &ToolResult{
		Output: fmt.Sprintf("Edited %s (%d replacement(s))", args.Path, replacements),
	}, nil
}
