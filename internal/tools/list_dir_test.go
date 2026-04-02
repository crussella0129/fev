// internal/tools/list_dir_test.go
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

func TestListDir_Basic(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.WriteFile(filepath.Join(dir, "file.go"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	tool := NewListDirTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "."})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "file.go") {
		t.Error("expected file.go in listing")
	}
	if !strings.Contains(result.Output, "subdir/") {
		t.Error("expected subdir/ in listing")
	}
}

func TestListDir_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, "visible.go"), []byte(""), 0644)

	tool := NewListDirTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"path": "."})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(result.Output, ".git") {
		t.Error("expected .git to be hidden")
	}
}
