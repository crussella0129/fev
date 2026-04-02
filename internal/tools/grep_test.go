// internal/tools/grep_test.go
package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func hasRipgrep() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

func TestGrep_BasicMatch(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep not installed")
	}

	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\t// TODO: implement\n}\n"), 0644)

	tool := NewGrepTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"pattern": "TODO",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "TODO") {
		t.Errorf("expected TODO in output, got %q", result.Output)
	}
}

func TestGrep_NoMatch(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep not installed")
	}

	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)

	tool := NewGrepTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"pattern": "NONEXISTENT_PATTERN_12345",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Output != "No matches found." {
		t.Errorf("expected 'No matches found.', got %q", result.Output)
	}
}

func TestGrep_WithGlob(t *testing.T) {
	if !hasRipgrep() {
		t.Skip("ripgrep not installed")
	}

	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main // match\n"), 0644)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("match here too\n"), 0644)

	tool := NewGrepTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"pattern": "match",
		"glob":    "*.go",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "main.go") {
		t.Error("expected main.go in results")
	}
	if strings.Contains(result.Output, "test.txt") {
		t.Error("expected test.txt to be excluded by glob filter")
	}
}
