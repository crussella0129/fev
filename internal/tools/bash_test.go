// internal/tools/bash_test.go
package tools

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestBash_SimpleCommand(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewBashTool(ws)

	// echo is a shell builtin, splitCommand wraps in sh -c / cmd /c
	args, _ := json.Marshal(map[string]interface{}{
		"command":     "echo hello",
		"description": "Print hello",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected 'hello' in output, got %q", result.Output)
	}
}

func TestBash_RequiresDescription(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewBashTool(ws)

	args, _ := json.Marshal(map[string]interface{}{
		"command": "echo test",
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error when description missing")
	}
}

func TestBash_BlocksDangerousCommand(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewBashTool(ws)

	args, _ := json.Marshal(map[string]interface{}{
		"command":     "rm -rf /",
		"description": "Delete everything",
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for dangerous command")
	}
}

func TestBash_BlocksInjection(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewBashTool(ws)

	injections := []string{
		"echo hello; rm -rf /",
		"echo $(whoami)",
		"echo `hostname`",
	}

	for _, cmd := range injections {
		t.Run(cmd, func(t *testing.T) {
			args, _ := json.Marshal(map[string]interface{}{
				"command":     cmd,
				"description": "test",
			})
			_, err := tool.Execute(context.Background(), args)
			if err == nil {
				t.Errorf("expected error for injection %q", cmd)
			}
		})
	}
}

func TestBash_Timeout(t *testing.T) {
	dir := t.TempDir()
	ws, _ := core.NewWorkspace(dir)
	tool := NewBashTool(ws)

	// Use a command that takes > 1s
	cmd := "sleep 10"
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1"
	}

	args, _ := json.Marshal(map[string]interface{}{
		"command":     cmd,
		"description": "Long-running command",
		"timeout":     1,
	})

	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
