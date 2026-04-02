// internal/tools/grep.go
package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type grepArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Glob    string `json:"glob,omitempty"`
	Context int    `json:"context,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// GrepTool searches file contents using ripgrep.
type GrepTool struct {
	workspace *core.Workspace
}

func NewGrepTool(ws *core.Workspace) *GrepTool {
	return &GrepTool{workspace: ws}
}

func (t *GrepTool) Name() string { return "grep" }
func (t *GrepTool) Description() string {
	return "Search file contents for a regex pattern. Uses ripgrep if available, otherwise native Go search."
}
func (t *GrepTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *GrepTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Regex pattern to search for"},
			"path": {"type": "string", "description": "Directory or file to search (default: workspace root)"},
			"glob": {"type": "string", "description": "File glob filter (e.g., '*.go')"},
			"context": {"type": "integer", "description": "Lines of context around matches"},
			"limit": {"type": "integer", "description": "Max results (default 250)"}
		},
		"required": ["pattern"]
	}`)
}

func (t *GrepTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args grepArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	searchPath := t.workspace.Root()
	if args.Path != "" {
		resolved, err := t.workspace.Resolve(args.Path)
		if err != nil {
			return nil, err
		}
		searchPath = resolved
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 250
	}

	// Try ripgrep first
	if rgPath, err := exec.LookPath("rg"); err == nil {
		return t.executeRipgrep(ctx, rgPath, args, searchPath, limit)
	}

	return t.executeNative(ctx, args, searchPath, limit)
}

func (t *GrepTool) executeRipgrep(ctx context.Context, rgPath string, args grepArgs, searchPath string, limit int) (*ToolResult, error) {
	cmdArgs := []string{
		"--no-heading",
		"--line-number",
		"--color", "never",
		"--max-count", fmt.Sprintf("%d", limit),
	}

	if args.Glob != "" {
		cmdArgs = append(cmdArgs, "--glob", args.Glob)
	}
	if args.Context > 0 {
		cmdArgs = append(cmdArgs, "--context", fmt.Sprintf("%d", args.Context))
	}

	cmdArgs = append(cmdArgs, args.Pattern, searchPath)

	cmd := exec.CommandContext(ctx, rgPath, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()

	if err != nil {
		// Exit code 1 = no matches (not an error)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &ToolResult{Output: "No matches found."}, nil
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("ripgrep error: %s", stderr.String())
		}
	}

	if strings.TrimSpace(output) == "" {
		return &ToolResult{Output: "No matches found."}, nil
	}

	return &ToolResult{Output: output}, nil
}

func (t *GrepTool) executeNative(ctx context.Context, args grepArgs, searchPath string, limit int) (*ToolResult, error) {
	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", args.Pattern, err)
	}

	var results []string
	count := 0

	walkErr := filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			name := ""
			if d != nil {
				name = d.Name()
			}
			if d != nil && d.IsDir() && (name == ".git" || name == "node_modules" || name == "vendor" || name == "__pycache__") {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return filepath.SkipAll
		}

		// Apply glob filter if specified
		if args.Glob != "" {
			matched, _ := filepath.Match(args.Glob, d.Name())
			if !matched {
				return nil
			}
		}

		// Skip binary/large files
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() > 5*1024*1024 {
			return nil
		}

		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		relPath, _ := filepath.Rel(searchPath, path)
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if count >= limit {
				return filepath.SkipAll
			}
			line := scanner.Text()
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d:%s", filepath.ToSlash(relPath), lineNum, line))
				count++
			}
		}
		return nil
	})

	if walkErr != nil && ctx.Err() == nil {
		return nil, fmt.Errorf("search error: %w", walkErr)
	}

	if len(results) == 0 {
		return &ToolResult{Output: "No matches found."}, nil
	}

	return &ToolResult{Output: strings.Join(results, "\n")}, nil
}
