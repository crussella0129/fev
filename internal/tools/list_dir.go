// internal/tools/list_dir.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"__pycache__": true, ".venv": true, "target": true,
}

type listDirArgs struct {
	Path  string `json:"path"`
	Depth int    `json:"depth,omitempty"`
}

// ListDirTool lists directory contents.
type ListDirTool struct {
	workspace *core.Workspace
}

func NewListDirTool(ws *core.Workspace) *ListDirTool {
	return &ListDirTool{workspace: ws}
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List directory contents. Ignores .git, node_modules, vendor." }
func (t *ListDirTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *ListDirTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Directory path (default: workspace root)"},
			"depth": {"type": "integer", "description": "Max depth (default: 2)"}
		},
		"required": ["path"]
	}`)
}

func (t *ListDirTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args listDirArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	depth := args.Depth
	if depth <= 0 {
		depth = 2
	}

	var lines []string
	walkDir(resolved, resolved, 0, depth, &lines)

	if len(lines) == 0 {
		return &ToolResult{Output: "Empty directory."}, nil
	}
	return &ToolResult{Output: strings.Join(lines, "\n")}, nil
}

func walkDir(root, dir string, currentDepth, maxDepth int, lines *[]string) {
	if currentDepth > maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if ignoredDirs[name] {
			continue
		}

		relPath, _ := filepath.Rel(root, filepath.Join(dir, name))
		relPath = filepath.ToSlash(relPath)

		if entry.IsDir() {
			*lines = append(*lines, relPath+"/")
			walkDir(root, filepath.Join(dir, name), currentDepth+1, maxDepth, lines)
		} else {
			*lines = append(*lines, relPath)
		}
	}
}
