// internal/tools/git_test.go
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

func initTestRepo(t *testing.T) (string, *core.Workspace) {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v: %v\n%s", args, err, out)
		}
	}

	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = dir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = dir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	ws, _ := core.NewWorkspace(dir)
	return dir, ws
}

func TestGit_Status(t *testing.T) {
	dir, ws := initTestRepo(t)
	os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main"), 0644)

	tool := NewGitTool(ws)
	args, _ := json.Marshal(map[string]interface{}{"operation": "status"})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "new.go") {
		t.Errorf("expected new.go in status, got %q", result.Output)
	}
}

func TestGit_AddAndCommit(t *testing.T) {
	dir, ws := initTestRepo(t)
	os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package main"), 0644)

	tool := NewGitTool(ws)

	// Add
	addArgs, _ := json.Marshal(map[string]interface{}{"operation": "add", "files": []string{"feature.go"}})
	_, err := tool.Execute(context.Background(), addArgs)
	if err != nil {
		t.Fatalf("git add: %v", err)
	}

	// Commit
	commitArgs, _ := json.Marshal(map[string]interface{}{"operation": "commit", "message": "add feature"})
	result, err := tool.Execute(context.Background(), commitArgs)
	if err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if result.Output == "" {
		t.Error("expected non-empty commit output")
	}
}

func TestGit_Log(t *testing.T) {
	_, ws := initTestRepo(t)
	tool := NewGitTool(ws)

	args, _ := json.Marshal(map[string]interface{}{"operation": "log", "count": 5})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "initial") {
		t.Errorf("expected 'initial' in log, got %q", result.Output)
	}
}

func TestGit_InvalidOperation(t *testing.T) {
	_, ws := initTestRepo(t)
	tool := NewGitTool(ws)

	args, _ := json.Marshal(map[string]interface{}{"operation": "push"})
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for unsupported operation")
	}
}
