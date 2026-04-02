// internal/tools/write_file.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFileTool creates or overwrites a file.
type WriteFileTool struct {
	workspace *core.Workspace
}

func NewWriteFileTool(ws *core.Workspace) *WriteFileTool {
	return &WriteFileTool{workspace: ws}
}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Create or overwrite a file. Creates parent directories if needed." }
func (t *WriteFileTool) PermissionTier() permission.PermissionTier { return permission.Write }

func (t *WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path (relative to workspace or absolute)"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args writeFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return nil, fmt.Errorf("create directories: %w", err)
	}

	// Check if file exists for diff reporting
	existed := false
	if _, err := os.Stat(resolved); err == nil {
		existed = true
	}

	if err := os.WriteFile(resolved, []byte(args.Content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	action := "Created"
	if existed {
		action = "Overwrote"
	}

	return &ToolResult{
		Output: fmt.Sprintf("%s %s (%d bytes)", action, args.Path, len(args.Content)),
	}, nil
}
