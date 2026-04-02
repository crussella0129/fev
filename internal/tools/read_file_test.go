// internal/tools/read_file_test.go
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

func TestReadFile_Basic(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	testFile := filepath.Join(dir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	tool := NewReadFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "test.go"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "package main") {
		t.Errorf("expected file contents, got %q", result.Output)
	}
}

func TestReadFile_WithLineNumbers(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)

	tool := NewReadFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "test.txt"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "1\t") {
		t.Error("expected line numbers in output")
	}
}

func TestReadFile_WithOffset(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	content := "line1\nline2\nline3\nline4\nline5\n"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644)

	tool := NewReadFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "test.txt", "offset": 2, "limit": 2})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "line3") {
		t.Errorf("expected line3 with offset=2, got %q", result.Output)
	}
}

func TestReadFile_OutOfBounds(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)

	tool := NewReadFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "../../etc/passwd"})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected workspace boundary error")
	}
}

func TestReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)

	tool := NewReadFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "nonexistent.go"})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
