// internal/tools/glob_test.go
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestGlob_MatchGoFiles(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0644)

	tool := NewGlobTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"pattern": "*.go"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "main.go") {
		t.Error("expected main.go in results")
	}
	if strings.Contains(result.Output, "readme.md") {
		t.Error("expected readme.md excluded")
	}
}

func TestGlob_RecursivePattern(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.go"), []byte(""), 0644)

	tool := NewGlobTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"pattern": "**/*.go"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "b.go") {
		t.Error("expected sub/b.go in recursive results")
	}
}

func TestGlob_NoMatches(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)

	tool := NewGlobTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"pattern": "*.xyz"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "No files") {
		t.Errorf("expected 'No files' message, got %q", result.Output)
	}
}
