// internal/tools/glob.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

const maxGlobResults = 100

type globArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// GlobTool discovers files by pattern.
type GlobTool struct {
	workspace *core.Workspace
}

func NewGlobTool(ws *core.Workspace) *GlobTool {
	return &GlobTool{workspace: ws}
}

func (t *GlobTool) Name() string        { return "glob" }
func (t *GlobTool) Description() string { return "Find files matching a glob pattern (e.g., '**/*.go'). Returns paths sorted by modification time." }
func (t *GlobTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *GlobTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Glob pattern (e.g., '**/*.go', 'src/*.ts')"},
			"path": {"type": "string", "description": "Base directory (default: workspace root)"}
		},
		"required": ["pattern"]
	}`)
}

func (t *GlobTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args globArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	basePath := t.workspace.Root()
	if args.Path != "" {
		resolved, err := t.workspace.Resolve(args.Path)
		if err != nil {
			return nil, err
		}
		basePath = resolved
	}

	type fileEntry struct {
		path    string
		modTime int64
	}

	var matches []fileEntry

	// Walk directory and match against pattern
	filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		// Skip hidden dirs and common noise
		name := d.Name()
		if d.IsDir() && (name == ".git" || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(basePath, path)
		relPath = filepath.ToSlash(relPath)

		matched, _ := filepath.Match(args.Pattern, filepath.Base(relPath))
		if !matched {
			// Try matching against full relative path for ** patterns
			matched, _ = filepath.Match(args.Pattern, relPath)
		}
		// Also try doublestar-style matching
		if !matched && strings.HasPrefix(args.Pattern, "**") {
			suffix := strings.TrimPrefix(args.Pattern, "**/")
			matched, _ = filepath.Match(suffix, filepath.Base(relPath))
		}

		if matched {
			info, infoErr := d.Info()
			modTime := int64(0)
			if infoErr == nil {
				modTime = info.ModTime().Unix()
			}
			matches = append(matches, fileEntry{path: relPath, modTime: modTime})
		}
		return nil
	})

	if len(matches) == 0 {
		return &ToolResult{Output: "No files matched the pattern."}, nil
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	// Truncate
	truncated := false
	if len(matches) > maxGlobResults {
		matches = matches[:maxGlobResults]
		truncated = true
	}

	var lines []string
	for _, m := range matches {
		lines = append(lines, m.path)
	}

	output := strings.Join(lines, "\n")
	if truncated {
		output += fmt.Sprintf("\n\n[truncated — showing %d of more results]", maxGlobResults)
	}

	return &ToolResult{Output: output, Truncated: truncated}, nil
}
