// internal/tools/read_file.go
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

const maxFileSize = 10 * 1024 * 1024 // 10MB

type readFileArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // 0-indexed line offset
	Limit  int    `json:"limit,omitempty"`  // max lines to return (0 = all)
}

// ReadFileTool reads file contents with line numbers.
type ReadFileTool struct {
	workspace *core.Workspace
}

func NewReadFileTool(ws *core.Workspace) *ReadFileTool {
	return &ReadFileTool{workspace: ws}
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read a file's contents with line numbers. Supports offset and limit for pagination." }
func (t *ReadFileTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path (relative to workspace or absolute)"},
			"offset": {"type": "integer", "description": "Start reading from this line (0-indexed)"},
			"limit": {"type": "integer", "description": "Maximum lines to return (0 = all, default 2000)"}
		},
		"required": ["path"]
	}`)
}

func (t *ReadFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args readFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", args.Path, err)
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file %q is too large (%d bytes, max %d)", args.Path, info.Size(), maxFileSize)
	}

	f, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= args.Offset {
			continue
		}
		if len(lines) >= limit {
			break
		}
		lines = append(lines, fmt.Sprintf("%d\t%s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{Output: output}, nil
}
