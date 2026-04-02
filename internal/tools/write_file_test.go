// internal/tools/write_file_test.go
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestWriteFile_Create(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewWriteFileTool(ws)

	args, _ := json.Marshal(map[string]interface{}{
		"path":    "new_file.go",
		"content": "package main\n\nfunc main() {}\n",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Output == "" {
		t.Error("expected non-empty output")
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(dir, "new_file.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "package main\n\nfunc main() {}\n" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestWriteFile_CreateSubdir(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewWriteFileTool(ws)

	args, _ := json.Marshal(map[string]interface{}{
		"path":    "sub/dir/file.txt",
		"content": "hello",
	})

	_, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "sub", "dir", "file.txt"))
	if string(content) != "hello" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestWriteFile_OutOfBounds(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewWriteFileTool(ws)

	args, _ := json.Marshal(map[string]interface{}{
		"path":    "../../etc/evil",
		"content": "bad",
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected workspace boundary error")
	}
}
