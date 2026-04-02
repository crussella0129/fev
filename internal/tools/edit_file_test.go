// internal/tools/edit_file_test.go
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

func TestEditFile_UniqueMatch(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	original := "package main\n\nfunc hello() {\n\treturn\n}\n"
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(original), 0644)

	tool := NewEditFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"path":       "main.go",
		"old_string": "func hello() {\n\treturn\n}",
		"new_string": "func hello() {\n\tfmt.Println(\"hello\")\n}",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Output == "" {
		t.Error("expected non-empty output")
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(content), "fmt.Println") {
		t.Errorf("expected edited content, got %q", string(content))
	}
}

func TestEditFile_NotUnique(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	original := "foo\nbar\nfoo\nbaz\n"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(original), 0644)

	tool := NewEditFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"path":       "test.txt",
		"old_string": "foo",
		"new_string": "qux",
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for non-unique match")
	}
	if !strings.Contains(err.Error(), "not unique") && !strings.Contains(err.Error(), "2 occurrences") {
		t.Errorf("expected 'not unique' error, got: %v", err)
	}
}

func TestEditFile_NoMatch(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	tool := NewEditFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"path":       "test.txt",
		"old_string": "nonexistent",
		"new_string": "replacement",
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestEditFile_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	original := "foo\nbar\nfoo\nbaz\n"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(original), 0644)

	tool := NewEditFileTool(ws)
	args, _ := json.Marshal(map[string]interface{}{
		"path":        "test.txt",
		"old_string":  "foo",
		"new_string":  "qux",
		"replace_all": true,
	})

	_, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "test.txt"))
	if strings.Contains(string(content), "foo") {
		t.Error("expected all 'foo' replaced")
	}
	if strings.Count(string(content), "qux") != 2 {
		t.Errorf("expected 2 'qux', got %d", strings.Count(string(content), "qux"))
	}
}
