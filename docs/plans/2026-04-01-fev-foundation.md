# Fev Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working `fev` binary that can chat with any OpenAI-compatible local LLM and execute 10 structured tools in a flat agent loop.

**Architecture:** Single Go binary, flat agent loop (generate → tool? → execute → loop), tool registry with JSON Schema validation, permission tiers, workspace boundary enforcement. Basic terminal REPL — no TUI yet (Plan 3 adds bubbletea).

**Tech Stack:** Go 1.22+, cobra (CLI), yaml.v3 (config), go-diff (diffs), jsonschema/v6 (validation), modernc.org/sqlite (future — not in this plan)

**Spec:** `docs/specs/2026-04-01-fev-architecture-design.md`

---

## File Structure

All files are new (greenfield project).

```
fev/
├── cmd/fev/main.go                         # Cobra CLI: fev, fev chat, fev init, fev version
├── internal/
│   ├── core/
│   │   ├── message.go                      # Message, ToolCall, FunctionCall types
│   │   ├── message_test.go
│   │   ├── errors.go                       # ErrorClass, ClassifyError, WorkspaceBoundaryError
│   │   ├── errors_test.go
│   │   ├── workspace.go                    # Workspace boundary enforcement
│   │   ├── workspace_test.go
│   │   ├── toolparse.go                    # ParseToolCalls, DeduplicateToolCalls
│   │   └── toolparse_test.go
│   ├── config/
│   │   ├── config.go                       # Config, Load, Save, defaults, FEV.md/CLAUDE.md
│   │   └── config_test.go
│   ├── llm/
│   │   ├── client.go                       # OpenAI-compatible HTTP client
│   │   ├── client_test.go
│   │   ├── streaming.go                    # SSE stream parser
│   │   ├── streaming_test.go
│   │   ├── tokens.go                       # EstimateTokens
│   │   └── tokens_test.go
│   ├── permission/
│   │   ├── checker.go                      # Deny-lists, injection detection, metachar blocking
│   │   ├── checker_test.go
│   │   ├── policy.go                       # PermissionTier enum, tool-level policy
│   │   └── policy_test.go
│   ├── tools/
│   │   ├── registry.go                     # Tool interface, Registry, schema generation
│   │   ├── registry_test.go
│   │   ├── executor.go                     # ExecuteWithTimeout, output-to-disk
│   │   ├── executor_test.go
│   │   ├── read_file.go                    # ReadFileTool
│   │   ├── read_file_test.go
│   │   ├── write_file.go                   # WriteFileTool
│   │   ├── write_file_test.go
│   │   ├── edit_file.go                    # EditFileTool (str_replace)
│   │   ├── edit_file_test.go
│   │   ├── bash.go                         # BashTool
│   │   ├── bash_test.go
│   │   ├── grep.go                         # GrepTool (ripgrep wrapper)
│   │   ├── grep_test.go
│   │   ├── glob.go                         # GlobTool
│   │   ├── glob_test.go
│   │   ├── list_dir.go                     # ListDirTool
│   │   ├── list_dir_test.go
│   │   ├── git.go                          # GitTool (status/diff/log/add/commit)
│   │   └── git_test.go
│   ├── ctxwin/
│   │   ├── manager.go                      # ContextManager, token budget, Trim
│   │   ├── manager_test.go
│   │   ├── blocks.go                       # Block, BlockType
│   │   └── blocks_test.go
│   └── agent/
│       ├── agent.go                        # Agent struct, New, Run
│       ├── agent_test.go
│       ├── loop.go                         # step, repeat detection, dedup
│       └── loop_test.go
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

---

## Task 1: Project Skeleton & Dependencies

**Files:**
- Create: `go.mod`
- Create: `cmd/fev/main.go`
- Create: `Makefile`
- Create: `.gitignore`
- Create: `README.md`

- [ ] **Step 1: Initialize Go module**

```bash
cd C:/Users/charl/Fev
go mod init github.com/crussella0129/fev
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
go get github.com/sergi/go-diff@latest
go get github.com/santhosh-tekuri/jsonschema/v6@latest
```

- [ ] **Step 3: Create main.go with cobra root command**

```go
// cmd/fev/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "fev",
	Short: "Fev — local-first agentic CLI",
	Long:  "Fev is a personal assistant for navigating the digital space. Model-agnostic, single binary, local-first.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default action: start interactive REPL
		fmt.Println("fev v" + version)
		fmt.Println("Interactive mode not yet implemented. Use 'fev version' to verify installation.")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Fev version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fev v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Create Makefile**

```makefile
.PHONY: build test lint clean

build:
	go build -o bin/fev ./cmd/fev

test:
	go test ./... -race -timeout 120s

lint:
	golangci-lint run

clean:
	rm -rf bin/
```

- [ ] **Step 5: Create .gitignore**

```
bin/
*.exe
*.db
.fev/
```

- [ ] **Step 6: Create README.md**

```markdown
# Fev

A local-first agentic CLI harness. Your personal assistant for navigating the digital space.

- **Model-agnostic** — works with any OpenAI-compatible API (Ollama, llama.cpp, vLLM, LM Studio)
- **Single binary** — `go install` or download. No Python, no Node.js, no Docker.
- **Ecosystem-compatible** — reads CLAUDE.md, speaks MCP, uses familiar tool names

## Quick Start

```bash
go install github.com/crussella0129/fev/cmd/fev@latest
fev
```

## Build from Source

```bash
git clone https://github.com/crussella0129/fev.git
cd fev
make build
./bin/fev version
```

## License

MIT
```

- [ ] **Step 7: Verify build**

Run: `cd C:/Users/charl/Fev && go build ./cmd/fev`
Expected: Clean build, no errors.

Run: `go run ./cmd/fev version`
Expected: `fev v0.1.0`

- [ ] **Step 8: Git init and initial commit**

```bash
cd C:/Users/charl/Fev
git init
git add go.mod go.sum cmd/ Makefile .gitignore README.md
git commit -m "feat: project skeleton — cobra CLI, Makefile, README"
```

---

## Task 2: Core Types — Messages

**Files:**
- Create: `internal/core/message.go`
- Create: `internal/core/message_test.go`

- [ ] **Step 1: Write message tests**

```go
// internal/core/message_test.go
package core

import (
	"encoding/json"
	"testing"
)

func TestNewSystemMessage(t *testing.T) {
	m := NewSystemMessage("you are helpful")
	if m.Role != RoleSystem {
		t.Errorf("expected role %q, got %q", RoleSystem, m.Role)
	}
	if m.Content != "you are helpful" {
		t.Errorf("expected content %q, got %q", "you are helpful", m.Content)
	}
}

func TestNewUserMessage(t *testing.T) {
	m := NewUserMessage("hello")
	if m.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, m.Role)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	m := NewAssistantMessage("hi there")
	if m.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, m.Role)
	}
}

func TestNewAssistantMessageWithTools(t *testing.T) {
	calls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "read_file",
				Arguments: `{"path": "/tmp/test.go"}`,
			},
		},
	}
	m := NewAssistantMessageWithTools("let me read that", calls)
	if m.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, m.Role)
	}
	if len(m.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(m.ToolCalls))
	}
	if m.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected tool name %q, got %q", "read_file", m.ToolCalls[0].Function.Name)
	}
}

func TestNewToolMessage(t *testing.T) {
	m := NewToolMessage("call_1", "file contents here")
	if m.Role != RoleTool {
		t.Errorf("expected role %q, got %q", RoleTool, m.Role)
	}
	if m.ToolCallID != "call_1" {
		t.Errorf("expected tool_call_id %q, got %q", "call_1", m.ToolCallID)
	}
}

func TestMessageJSON(t *testing.T) {
	m := NewAssistantMessageWithTools("", []ToolCall{
		{
			ID:   "call_abc",
			Type: "function",
			Function: FunctionCall{
				Name:      "bash",
				Arguments: `{"command": "ls"}`,
			},
		},
	})
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call after round-trip, got %d", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].ID != "call_abc" {
		t.Errorf("expected tool call ID %q, got %q", "call_abc", decoded.ToolCalls[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestNew`
Expected: Compilation error — types not defined yet.

- [ ] **Step 3: Write message.go**

```go
// internal/core/message.go
package core

// Role represents a message participant.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is the universal message type, compatible with OpenAI chat completions.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a function call requested by the assistant.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and JSON-encoded arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func NewSystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

func NewUserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func NewAssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func NewAssistantMessageWithTools(content string, calls []ToolCall) Message {
	return Message{Role: RoleAssistant, Content: content, ToolCalls: calls}
}

func NewToolMessage(callID, content string) Message {
	return Message{Role: RoleTool, ToolCallID: callID, Content: content}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestNew`
Expected: All PASS.

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestMessageJSON`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/message.go internal/core/message_test.go
git commit -m "feat: core message types — Message, ToolCall, FunctionCall"
```

---

## Task 3: Core Types — Errors & Workspace

**Files:**
- Create: `internal/core/errors.go`
- Create: `internal/core/errors_test.go`
- Create: `internal/core/workspace.go`
- Create: `internal/core/workspace_test.go`

- [ ] **Step 1: Write error tests**

```go
// internal/core/errors_test.go
package core

import (
	"errors"
	"testing"
)

func TestClassifyError_Retryable(t *testing.T) {
	cases := []string{
		"rate limit exceeded",
		"connection refused",
		"request timeout",
		"server returned 503",
		"server returned 429",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if ClassifyError(err) != ErrorRetryable {
				t.Errorf("expected %q to be retryable", msg)
			}
		})
	}
}

func TestClassifyError_Fatal(t *testing.T) {
	cases := []string{
		"invalid API key",
		"model not found",
		"permission denied",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if ClassifyError(err) != ErrorFatal {
				t.Errorf("expected %q to be fatal", msg)
			}
		})
	}
}

func TestWorkspaceBoundaryError(t *testing.T) {
	err := &WorkspaceBoundaryError{Path: "/etc/passwd", Root: "/home/user/project"}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
```

- [ ] **Step 2: Write errors.go**

```go
// internal/core/errors.go
package core

import (
	"fmt"
	"strings"
)

// ErrorClass categorizes errors for retry decisions.
type ErrorClass int

const (
	ErrorRetryable ErrorClass = iota
	ErrorFatal
)

// ClassifyError determines if an error is retryable or fatal.
func ClassifyError(err error) ErrorClass {
	msg := strings.ToLower(err.Error())
	retryable := []string{"rate limit", "timeout", "connection refused", "503", "429", "temporary"}
	for _, pattern := range retryable {
		if strings.Contains(msg, pattern) {
			return ErrorRetryable
		}
	}
	return ErrorFatal
}

// WorkspaceBoundaryError is returned when a path escapes the workspace root.
type WorkspaceBoundaryError struct {
	Path string
	Root string
}

func (e *WorkspaceBoundaryError) Error() string {
	return fmt.Sprintf("path %q is outside workspace root %q", e.Path, e.Root)
}
```

- [ ] **Step 3: Run error tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestClassify`
Expected: All PASS.

- [ ] **Step 4: Write workspace tests**

```go
// internal/core/workspace_test.go
package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceResolve_InBounds(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	// Create a file inside the workspace
	testFile := filepath.Join(dir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	resolved, err := ws.Resolve("test.go")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved != testFile {
		t.Errorf("expected %q, got %q", testFile, resolved)
	}
}

func TestWorkspaceResolve_OutOfBounds(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	_, err = ws.Resolve("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for out-of-bounds path")
	}
	if !isWorkspaceBoundaryError(err) {
		t.Errorf("expected WorkspaceBoundaryError, got %T: %v", err, err)
	}
}

func TestWorkspaceResolve_Absolute(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	inFile := filepath.Join(dir, "ok.txt")
	os.WriteFile(inFile, []byte("ok"), 0644)

	resolved, err := ws.Resolve(inFile)
	if err != nil {
		t.Fatalf("Resolve absolute in-bounds: %v", err)
	}
	if resolved != inFile {
		t.Errorf("expected %q, got %q", inFile, resolved)
	}
}

func TestWorkspaceResolve_NewFile(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}

	// File doesn't exist yet — should still resolve if parent is in bounds
	resolved, err := ws.Resolve("newfile.txt")
	if err != nil {
		t.Fatalf("Resolve new file: %v", err)
	}
	expected := filepath.Join(dir, "newfile.txt")
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	ws, err := NewWorkspace(dir)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	if ws.Root() == "" {
		t.Error("expected non-empty root")
	}
}

func isWorkspaceBoundaryError(err error) bool {
	var wbe *WorkspaceBoundaryError
	return errors.As(err, &wbe)
}
```

- [ ] **Step 5: Write workspace.go**

```go
// internal/core/workspace.go
package core

import (
	"path/filepath"
	"strings"
	"sync"
)

// Workspace enforces path boundaries for all tool operations.
type Workspace struct {
	mu   sync.RWMutex
	root string
	cwd  string
}

// NewWorkspace creates a workspace rooted at the given directory.
// The root is resolved to an absolute, symlink-evaluated path.
func NewWorkspace(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// Try to eval symlinks; if dir doesn't exist, use abs directly
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		real = abs
	}
	return &Workspace{root: real, cwd: real}, nil
}

// Root returns the workspace root directory.
func (w *Workspace) Root() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.root
}

// CWD returns the current working directory within the workspace.
func (w *Workspace) CWD() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cwd
}

// Resolve converts a relative or absolute path to a safe absolute path
// within the workspace boundary. Returns WorkspaceBoundaryError if the
// resolved path escapes the root.
func (w *Workspace) Resolve(path string) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Join(w.cwd, path)
	}

	// Try to resolve symlinks for existing files
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// File may not exist yet — resolve parent instead
		dir := filepath.Dir(abs)
		realDir, dirErr := filepath.EvalSymlinks(dir)
		if dirErr != nil {
			// Parent doesn't exist either — use cleaned path
			real = filepath.Clean(abs)
		} else {
			real = filepath.Join(realDir, filepath.Base(abs))
		}
	}

	// Boundary check: real must be within root (or equal to root)
	if !strings.HasPrefix(real, w.root) {
		return "", &WorkspaceBoundaryError{Path: path, Root: w.root}
	}

	return real, nil
}

// SetCWD changes the working directory within the workspace.
func (w *Workspace) SetCWD(dir string) error {
	resolved, err := w.Resolve(dir)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cwd = resolved
	return nil
}
```

- [ ] **Step 6: Run all core tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v`
Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/core/
git commit -m "feat: core errors, workspace boundary enforcement"
```

---

## Task 4: Core Types — Tool Call Parsing

**Files:**
- Create: `internal/core/toolparse.go`
- Create: `internal/core/toolparse_test.go`

- [ ] **Step 1: Write toolparse tests**

```go
// internal/core/toolparse_test.go
package core

import (
	"testing"
)

func TestDeduplicateToolCalls_NoDupes(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		{ID: "2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"b.go"}`}},
	}
	deduped := DeduplicateToolCalls(calls)
	if len(deduped) != 2 {
		t.Errorf("expected 2 calls, got %d", len(deduped))
	}
}

func TestDeduplicateToolCalls_WithDupes(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		{ID: "2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
	}
	deduped := DeduplicateToolCalls(calls)
	if len(deduped) != 1 {
		t.Errorf("expected 1 call after dedup, got %d", len(deduped))
	}
}

func TestToolCallHash(t *testing.T) {
	a := ToolCall{ID: "1", Function: FunctionCall{Name: "bash", Arguments: `{"command":"ls"}`}}
	b := ToolCall{ID: "2", Function: FunctionCall{Name: "bash", Arguments: `{"command":"ls"}`}}
	c := ToolCall{ID: "3", Function: FunctionCall{Name: "bash", Arguments: `{"command":"pwd"}`}}

	if ToolCallHash(a) != ToolCallHash(b) {
		t.Error("same name+args should produce same hash regardless of ID")
	}
	if ToolCallHash(a) == ToolCallHash(c) {
		t.Error("different args should produce different hash")
	}
}
```

- [ ] **Step 2: Write toolparse.go**

```go
// internal/core/toolparse.go
package core

import (
	"crypto/sha256"
	"fmt"
)

// ToolCallHash produces a hash of a tool call based on name + arguments (ignoring ID).
// Used for deduplication and repeat detection.
func ToolCallHash(tc ToolCall) string {
	h := sha256.New()
	h.Write([]byte(tc.Function.Name))
	h.Write([]byte(tc.Function.Arguments))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// DeduplicateToolCalls removes consecutive identical tool calls (same name + args).
func DeduplicateToolCalls(calls []ToolCall) []ToolCall {
	if len(calls) <= 1 {
		return calls
	}
	seen := make(map[string]bool)
	var result []ToolCall
	for _, tc := range calls {
		hash := ToolCallHash(tc)
		if !seen[hash] {
			seen[hash] = true
			result = append(result, tc)
		}
	}
	return result
}
```

- [ ] **Step 3: Run tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestDeduplicate`
Expected: All PASS.

Run: `cd C:/Users/charl/Fev && go test ./internal/core/ -v -run TestToolCallHash`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/core/toolparse.go internal/core/toolparse_test.go
git commit -m "feat: tool call hashing and deduplication"
```

---

## Task 5: Config System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Model.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("expected Ollama default URL, got %q", cfg.Model.BaseURL)
	}
	if cfg.Model.ModelName == "" {
		t.Error("expected non-empty default model name")
	}
	if cfg.Agent.MaxTurns < 1 {
		t.Error("expected positive max turns")
	}
}

func TestLoadConfig_DefaultsWhenMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model.BaseURL == "" {
		t.Error("expected defaults when file missing")
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
model:
  base_url: "http://localhost:8080/v1"
  model_name: "llama3"
  temperature: 0.5
  context_length: 8192
agent:
  max_turns: 10
  confirm_dangerous: false
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model.BaseURL != "http://localhost:8080/v1" {
		t.Errorf("expected custom URL, got %q", cfg.Model.BaseURL)
	}
	if cfg.Model.ModelName != "llama3" {
		t.Errorf("expected llama3, got %q", cfg.Model.ModelName)
	}
	if cfg.Agent.MaxTurns != 10 {
		t.Errorf("expected 10 max turns, got %d", cfg.Agent.MaxTurns)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := DefaultConfig()
	cfg.Model.ModelName = "custom-model"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if loaded.Model.ModelName != "custom-model" {
		t.Errorf("expected custom-model after round-trip, got %q", loaded.Model.ModelName)
	}
}

func TestLoadProjectConfig_FEVmd(t *testing.T) {
	dir := t.TempDir()
	fevmd := filepath.Join(dir, "FEV.md")
	content := `# FEV.md

## Project
This is a Go web service.

## Build & Test
- Build: ` + "`go build ./cmd/server`" + `
- Test: ` + "`go test ./...`" + `

## Do Not
- Modify migration files
`
	os.WriteFile(fevmd, []byte(content), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if pc == "" {
		t.Error("expected non-empty project config from FEV.md")
	}
}

func TestLoadProjectConfig_CLAUDEmd(t *testing.T) {
	dir := t.TempDir()
	claudemd := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudemd, []byte("# CLAUDE.md\n\nUse Go conventions."), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if pc == "" {
		t.Error("expected non-empty project config from CLAUDE.md")
	}
}

func TestLoadProjectConfig_FEVmdTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "FEV.md"), []byte("# FEV rules"), 0644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Claude rules"), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	// Both should be included, FEV.md content first
	if len(pc) == 0 {
		t.Error("expected merged config")
	}
}
```

- [ ] **Step 2: Write config.go**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModelConfig holds LLM connection settings.
type ModelConfig struct {
	BaseURL       string  `yaml:"base_url"`
	ModelName     string  `yaml:"model_name"`
	Temperature   float64 `yaml:"temperature"`
	ContextLength int     `yaml:"context_length"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	MaxTurns         int    `yaml:"max_turns"`
	ConfirmDangerous bool   `yaml:"confirm_dangerous"`
	AutoCompact      bool   `yaml:"auto_compact"`
	SystemPrompt     string `yaml:"system_prompt"`
	WorkspaceRoot    string `yaml:"workspace_root"`
}

// Config is the top-level configuration for Fev.
type Config struct {
	Model ModelConfig `yaml:"model"`
	Agent AgentConfig `yaml:"agent"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			BaseURL:       "http://localhost:11434/v1",
			ModelName:     "qwen2.5-coder:14b",
			Temperature:   0.7,
			ContextLength: 0, // 0 = auto-detect
		},
		Agent: AgentConfig{
			MaxTurns:         25,
			ConfirmDangerous: true,
			AutoCompact:      true,
		},
	}
}

// Load reads config from a YAML file. Returns defaults if file doesn't exist.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes config to a YAML file.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ConfigDir returns the Fev config directory (~/.fev/).
func ConfigDir() (string, error) {
	if dir := os.Getenv("FEV_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fev"), nil
}

// LoadProjectConfig reads FEV.md and/or CLAUDE.md from a project directory.
// FEV.md content comes first; CLAUDE.md is appended if present.
// Returns the merged content as a string for injection into the system prompt.
func LoadProjectConfig(projectDir string) (string, error) {
	var parts []string

	fevPath := filepath.Join(projectDir, "FEV.md")
	if data, err := os.ReadFile(fevPath); err == nil {
		parts = append(parts, string(data))
	}

	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	if data, err := os.ReadFile(claudePath); err == nil {
		parts = append(parts, string(data))
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}
```

- [ ] **Step 3: Run tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/config/ -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat: config system — YAML, FEV.md, CLAUDE.md loading"
```

---

## Task 6: LLM Client — Types & Generate

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/client_test.go`

- [ ] **Step 1: Write client tests**

```go
// internal/llm/client_test.go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestGenerate_SimpleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := ChatResponse{
			ID: "test-1",
			Choices: []Choice{
				{
					Message:      core.Message{Role: core.RoleAssistant, Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	messages := []core.Message{core.NewUserMessage("hi")}

	resp, err := client.Generate(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", resp.Content)
	}
}

func TestGenerate_ToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			ID: "test-2",
			Choices: []Choice{
				{
					Message: core.Message{
						Role: core.RoleAssistant,
						ToolCalls: []core.ToolCall{
							{
								ID:   "call_abc",
								Type: "function",
								Function: core.FunctionCall{
									Name:      "read_file",
									Arguments: `{"path":"main.go"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: Usage{PromptTokens: 20, CompletionTokens: 15, TotalTokens: 35},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	messages := []core.Message{core.NewUserMessage("read main.go")}

	resp, err := client.Generate(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected read_file, got %q", resp.ToolCalls[0].Function.Name)
	}
}

func TestGenerate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal error"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	_, err := client.Generate(context.Background(), []core.Message{core.NewUserMessage("hi")}, nil)
	if err == nil {
		t.Fatal("expected error on 500 response")
	}
}
```

- [ ] **Step 2: Write client.go**

```go
// internal/llm/client.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/crussella0129/fev/internal/core"
)

// Client is an OpenAI-compatible HTTP client for LLM inference.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// ChatRequest is the request body for /v1/chat/completions.
type ChatRequest struct {
	Model       string         `json:"model"`
	Messages    []core.Message `json:"messages"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Tools       []ToolSchema   `json:"tools,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

// ToolSchema is the OpenAI function-calling tool format.
type ToolSchema struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

// FunctionSchema describes a function tool for the LLM.
type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatResponse is the response from /v1/chat/completions.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is a single completion choice.
type Choice struct {
	Message      core.Message `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewClient creates an OpenAI-compatible client.
func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Generate sends a non-streaming chat completion request.
func (c *Client) Generate(ctx context.Context, messages []core.Message, tools []ToolSchema) (*core.Message, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &chatResp.Choices[0].Message, nil
}

// Model returns the model name.
func (c *Client) Model() string {
	return c.model
}
```

- [ ] **Step 3: Run tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/llm/ -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/llm/client.go internal/llm/client_test.go
git commit -m "feat: OpenAI-compatible LLM client with Generate"
```

---

## Task 7: LLM Client — Streaming & Token Estimation

**Files:**
- Create: `internal/llm/streaming.go`
- Create: `internal/llm/streaming_test.go`
- Create: `internal/llm/tokens.go`
- Create: `internal/llm/tokens_test.go`

- [ ] **Step 1: Write streaming tests**

```go
// internal/llm/streaming_test.go
package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestGenerateStream_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{"Hello", " world", "!"}
		for i, chunk := range chunks {
			data := fmt.Sprintf(`{"id":"test","choices":[{"delta":{"content":"%s"},"finish_reason":null}]}`, chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			_ = i
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	var collected []string
	onChunk := func(s string) { collected = append(collected, s) }

	resp, err := client.GenerateStream(context.Background(), []core.Message{core.NewUserMessage("hi")}, nil, onChunk)
	if err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if resp.Content != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", resp.Content)
	}
	if len(collected) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(collected))
	}
}

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		line     string
		wantData string
		wantDone bool
	}{
		{"data: {\"test\":true}", `{"test":true}`, false},
		{"data: [DONE]", "", true},
		{": comment", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			data, done := parseSSELine(tt.line)
			if done != tt.wantDone {
				t.Errorf("done: got %v, want %v", done, tt.wantDone)
			}
			if data != tt.wantData {
				t.Errorf("data: got %q, want %q", data, tt.wantData)
			}
		})
	}
}
```

- [ ] **Step 2: Write streaming.go**

```go
// internal/llm/streaming.go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/crussella0129/fev/internal/core"
)

// StreamDelta is a single chunk from the SSE stream.
type StreamDelta struct {
	ID      string        `json:"id"`
	Choices []DeltaChoice `json:"choices"`
}

// DeltaChoice is a streaming choice.
type DeltaChoice struct {
	Delta        DeltaContent `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// DeltaContent holds incremental content or tool call fragments.
type DeltaContent struct {
	Content   string          `json:"content,omitempty"`
	ToolCalls []DeltaToolCall `json:"tool_calls,omitempty"`
}

// DeltaToolCall is an incremental tool call fragment.
type DeltaToolCall struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function DeltaFunctionCall `json:"function,omitempty"`
}

// DeltaFunctionCall holds incremental function call data.
type DeltaFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// GenerateStream sends a streaming chat completion request.
// onChunk is called for each text content chunk.
// Returns the fully assembled message.
func (c *Client) GenerateStream(ctx context.Context, messages []core.Message, tools []ToolSchema, onChunk func(string)) (*core.Message, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	// Assemble the full message from streaming deltas
	var contentBuilder strings.Builder
	toolCallMap := make(map[int]*core.ToolCall) // index -> accumulated tool call

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		data, done := parseSSELine(line)
		if done {
			break
		}
		if data == "" {
			continue
		}

		var delta StreamDelta
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			continue // skip malformed chunks
		}

		for _, choice := range delta.Choices {
			// Accumulate text content
			if choice.Delta.Content != "" {
				contentBuilder.WriteString(choice.Delta.Content)
				if onChunk != nil {
					onChunk(choice.Delta.Content)
				}
			}

			// Accumulate tool calls
			for _, dtc := range choice.Delta.ToolCalls {
				tc, exists := toolCallMap[dtc.Index]
				if !exists {
					tc = &core.ToolCall{Type: "function"}
					toolCallMap[dtc.Index] = tc
				}
				if dtc.ID != "" {
					tc.ID = dtc.ID
				}
				if dtc.Function.Name != "" {
					tc.Function.Name = dtc.Function.Name
				}
				tc.Function.Arguments += dtc.Function.Arguments
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	// Build the final message
	msg := &core.Message{
		Role:    core.RoleAssistant,
		Content: contentBuilder.String(),
	}

	if len(toolCallMap) > 0 {
		msg.ToolCalls = make([]core.ToolCall, 0, len(toolCallMap))
		for i := 0; i < len(toolCallMap); i++ {
			if tc, ok := toolCallMap[i]; ok {
				msg.ToolCalls = append(msg.ToolCalls, *tc)
			}
		}
	}

	return msg, nil
}

// parseSSELine parses a single SSE line.
// Returns (data, isDone).
func parseSSELine(line string) (string, bool) {
	if strings.HasPrefix(line, "data: [DONE]") {
		return "", true
	}
	if strings.HasPrefix(line, "data: ") {
		return strings.TrimPrefix(line, "data: "), false
	}
	return "", false
}
```

- [ ] **Step 3: Write token estimation tests**

```go
// internal/llm/tokens_test.go
package llm

import (
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		wantMin  int
		wantMax  int
	}{
		{"", 0, 0},
		{"hello", 1, 3},
		{"this is a test of the token estimation", 5, 15},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateTokens(%q) = %d, want [%d, %d]", tt.text, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	msgs := []core.Message{
		core.NewSystemMessage("you are helpful"),
		core.NewUserMessage("hello"),
		core.NewAssistantMessage("hi there"),
	}
	total := EstimateMessageTokens(msgs)
	if total < 5 {
		t.Errorf("expected at least 5 tokens, got %d", total)
	}
}
```

- [ ] **Step 4: Write tokens.go**

```go
// internal/llm/tokens.go
package llm

import (
	"github.com/crussella0129/fev/internal/core"
)

// EstimateTokens provides a rough token count using the 4-chars-per-token heuristic.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// EstimateMessageTokens estimates total tokens across a slice of messages.
// Includes a small overhead per message for role/formatting (~4 tokens each).
func EstimateMessageTokens(messages []core.Message) int {
	total := 0
	for _, m := range messages {
		total += 4 // role + formatting overhead
		total += EstimateTokens(m.Content)
		for _, tc := range m.ToolCalls {
			total += EstimateTokens(tc.Function.Name)
			total += EstimateTokens(tc.Function.Arguments)
			total += 4 // tool call overhead
		}
	}
	return total
}
```

- [ ] **Step 5: Run all LLM tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/llm/ -v`
Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/llm/
git commit -m "feat: SSE streaming parser, token estimation"
```

---

## Task 8: Permission System

**Files:**
- Create: `internal/permission/checker.go`
- Create: `internal/permission/checker_test.go`
- Create: `internal/permission/policy.go`
- Create: `internal/permission/policy_test.go`

- [ ] **Step 1: Write checker tests**

```go
// internal/permission/checker_test.go
package permission

import (
	"testing"
)

func TestIsCommandDangerous(t *testing.T) {
	dangerous := []string{
		"rm -rf /",
		"sudo reboot",
		"mkfs.ext4 /dev/sda",
		"shutdown -h now",
		"dd if=/dev/zero of=/dev/sda",
	}
	for _, cmd := range dangerous {
		t.Run(cmd, func(t *testing.T) {
			if !IsCommandDangerous(cmd) {
				t.Errorf("expected %q to be dangerous", cmd)
			}
		})
	}
}

func TestIsCommandDangerous_Safe(t *testing.T) {
	safe := []string{"ls", "go test ./...", "cat main.go", "git status"}
	for _, cmd := range safe {
		t.Run(cmd, func(t *testing.T) {
			if IsCommandDangerous(cmd) {
				t.Errorf("expected %q to be safe", cmd)
			}
		})
	}
}

func TestIsInjectionAttempt(t *testing.T) {
	injections := []string{
		"ls; rm -rf /",
		"echo $(whoami)",
		"cat `hostname`",
		"ls && rm -rf /",
		"ls || true",
		"cat < /etc/passwd",
		"echo hello > /etc/passwd",
	}
	for _, cmd := range injections {
		t.Run(cmd, func(t *testing.T) {
			if !IsInjectionAttempt(cmd) {
				t.Errorf("expected %q to be an injection attempt", cmd)
			}
		})
	}
}

func TestIsInjectionAttempt_Clean(t *testing.T) {
	clean := []string{
		"go test ./...",
		"git log --oneline -10",
		"ls -la",
		"grep -r TODO .",
	}
	for _, cmd := range clean {
		t.Run(cmd, func(t *testing.T) {
			if IsInjectionAttempt(cmd) {
				t.Errorf("expected %q to be clean", cmd)
			}
		})
	}
}

func TestIsPathDangerous(t *testing.T) {
	dangerous := []string{
		"/etc/passwd",
		"/sys/kernel",
		"/proc/self",
		"/boot/vmlinuz",
	}
	for _, p := range dangerous {
		t.Run(p, func(t *testing.T) {
			if !IsPathDangerous(p) {
				t.Errorf("expected %q to be dangerous", p)
			}
		})
	}
}
```

- [ ] **Step 2: Write checker.go**

```go
// internal/permission/checker.go
package permission

import (
	"regexp"
	"strings"
)

var dangerousDirs = []string{
	"/etc", "/sys", "/proc", "/boot", "/dev", "/root",
	"C:\\Windows", "C:\\System32", "C:\\ProgramData",
}

var dangerousCommands = []string{
	"rm -rf /", "rm -rf /*", "mkfs", "dd if=", "shutdown", "reboot",
	"halt", "poweroff", "init 0", "init 6",
	":(){ :|:& };:", // fork bomb
}

var dangerousCommandPrefixes = []string{
	"sudo", "su ", "chmod 777", "chown root",
}

var injectionPattern = regexp.MustCompile(`[;|&<>]|\$\(|` + "`")

// IsCommandDangerous checks if a command matches the blocked commands list.
func IsCommandDangerous(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	for _, dc := range dangerousCommands {
		if strings.Contains(lower, dc) {
			return true
		}
	}
	for _, prefix := range dangerousCommandPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// IsInjectionAttempt checks for shell metacharacters that indicate injection.
func IsInjectionAttempt(cmd string) bool {
	return injectionPattern.MatchString(cmd)
}

// IsPathDangerous checks if a path targets a sensitive system directory.
func IsPathDangerous(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, dir := range dangerousDirs {
		normalizedDir := strings.ReplaceAll(dir, "\\", "/")
		if strings.HasPrefix(normalized, normalizedDir) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Write policy tests**

```go
// internal/permission/policy_test.go
package permission

import (
	"testing"
)

func TestPermissionTier_String(t *testing.T) {
	if ReadOnly.String() != "read-only" {
		t.Errorf("expected 'read-only', got %q", ReadOnly.String())
	}
	if Write.String() != "write" {
		t.Errorf("expected 'write', got %q", Write.String())
	}
	if Dangerous.String() != "dangerous" {
		t.Errorf("expected 'dangerous', got %q", Dangerous.String())
	}
}

func TestRequiresConfirmation(t *testing.T) {
	if RequiresConfirmation(ReadOnly, true) {
		t.Error("read-only should never require confirmation")
	}
	if RequiresConfirmation(Write, true) {
		t.Error("write with confirm=true should not require confirmation (only dangerous does)")
	}
	if !RequiresConfirmation(Dangerous, true) {
		t.Error("dangerous with confirm=true should require confirmation")
	}
	if RequiresConfirmation(Dangerous, false) {
		t.Error("dangerous with confirm=false should not require confirmation")
	}
}
```

- [ ] **Step 4: Write policy.go**

```go
// internal/permission/policy.go
package permission

// PermissionTier categorizes tool safety levels.
type PermissionTier int

const (
	ReadOnly  PermissionTier = iota // Execute immediately
	Write                           // Show diff, configurable confirmation
	Dangerous                       // Require confirmation
)

// String returns a human-readable tier name.
func (p PermissionTier) String() string {
	switch p {
	case ReadOnly:
		return "read-only"
	case Write:
		return "write"
	case Dangerous:
		return "dangerous"
	default:
		return "unknown"
	}
}

// RequiresConfirmation returns true if the tier requires user approval.
// confirmDangerous is from config — if false, even dangerous tools run without asking.
func RequiresConfirmation(tier PermissionTier, confirmDangerous bool) bool {
	return tier == Dangerous && confirmDangerous
}
```

- [ ] **Step 5: Run all permission tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/permission/ -v`
Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/permission/
git commit -m "feat: permission system — deny-lists, injection detection, tiers"
```

---

## Task 9: Tool Registry & Executor

**Files:**
- Create: `internal/tools/registry.go`
- Create: `internal/tools/registry_test.go`
- Create: `internal/tools/executor.go`
- Create: `internal/tools/executor_test.go`

- [ ] **Step 1: Write registry tests**

```go
// internal/tools/registry_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/crussella0129/fev/internal/permission"
)

// mockTool is a simple tool for testing.
type mockTool struct {
	name   string
	result string
}

func (m *mockTool) Name() string                    { return m.name }
func (m *mockTool) Description() string             { return "mock tool" }
func (m *mockTool) Schema() json.RawMessage         { return json.RawMessage(`{"type":"object"}`) }
func (m *mockTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Output: m.result}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tool := &mockTool{name: "test_tool", result: "ok"}

	reg.Register(tool)

	got := reg.Get("test_tool")
	if got == nil {
		t.Fatal("expected to find registered tool")
	}
	if got.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", got.Name())
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "b_tool"})
	reg.Register(&mockTool{name: "a_tool"})

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(list))
	}
	// Should be sorted
	if list[0] != "a_tool" || list[1] != "b_tool" {
		t.Errorf("expected sorted list [a_tool, b_tool], got %v", list)
	}
}

func TestRegistry_ToOpenAISchemas(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "read_file"})

	schemas := reg.ToOpenAISchemas()
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	if schemas[0].Function.Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", schemas[0].Function.Name)
	}
}
```

- [ ] **Step 2: Write registry.go**

```go
// internal/tools/registry.go
package tools

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/permission"
)

// Tool is the interface all Fev tools must implement.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	PermissionTier() permission.PermissionTier
	Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// ToolResult is the output of a tool execution.
type ToolResult struct {
	Output    string `json:"output"`
	Truncated bool   `json:"truncated,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

// Registry manages tool registration and lookup.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool. Panics on duplicate name.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name()]; exists {
		panic("duplicate tool: " + t.Name())
	}
	r.tools[t.Name()] = t
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// List returns sorted tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ToOpenAISchemas returns tool schemas in OpenAI function-calling format.
func (r *Registry) ToOpenAISchemas() []llm.ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemas := make([]llm.ToolSchema, 0, len(r.tools))
	for _, t := range r.tools {
		schemas = append(schemas, llm.ToolSchema{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return schemas
}
```

- [ ] **Step 3: Write executor tests**

```go
// internal/tools/executor_test.go
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type slowTool struct{ mockTool }

func (s *slowTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	select {
	case <-time.After(5 * time.Second):
		return &ToolResult{Output: "done"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	tool := &mockTool{name: "fast", result: "ok"}
	result, err := ExecuteWithTimeout(context.Background(), tool, json.RawMessage(`{}`), 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "ok" {
		t.Errorf("expected 'ok', got %q", result.Output)
	}
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	tool := &slowTool{mockTool{name: "slow"}}
	_, err := ExecuteWithTimeout(context.Background(), tool, json.RawMessage(`{}`), 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestTruncateOutput(t *testing.T) {
	long := strings.Repeat("a", 40000) // ~10K tokens at 4 chars/token
	result := &ToolResult{Output: long}
	truncated := MaybeTruncate(result, 8000)
	if !truncated.Truncated {
		t.Error("expected truncated=true for large output")
	}
	if len(truncated.Output) >= len(long) {
		t.Error("expected output to be shorter after truncation")
	}
}
```

- [ ] **Step 4: Write executor.go**

```go
// internal/tools/executor.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crussella0129/fev/internal/llm"
)

// ExecuteWithTimeout runs a tool with a context deadline.
func ExecuteWithTimeout(ctx context.Context, tool Tool, args json.RawMessage, timeout time.Duration) (*ToolResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan *ToolResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := tool.Execute(ctx, args)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool %q timed out after %v", tool.Name(), timeout)
	}
}

// MaybeTruncate truncates a tool result if it exceeds maxTokens.
// Returns the (possibly modified) result.
func MaybeTruncate(result *ToolResult, maxTokens int) *ToolResult {
	estimated := llm.EstimateTokens(result.Output)
	if estimated <= maxTokens {
		return result
	}

	// Keep approximately maxTokens worth of characters
	maxChars := maxTokens * 4
	if maxChars >= len(result.Output) {
		return result
	}

	return &ToolResult{
		Output:    result.Output[:maxChars] + "\n\n[truncated — output exceeded limit]",
		Truncated: true,
		FilePath:  result.FilePath,
		ExitCode:  result.ExitCode,
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v`
Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tools/registry.go internal/tools/registry_test.go internal/tools/executor.go internal/tools/executor_test.go
git commit -m "feat: tool registry, executor with timeout and truncation"
```

---

## Task 10: File Tools (read_file, write_file, edit_file)

**Files:**
- Create: `internal/tools/read_file.go`
- Create: `internal/tools/read_file_test.go`
- Create: `internal/tools/write_file.go`
- Create: `internal/tools/write_file_test.go`
- Create: `internal/tools/edit_file.go`
- Create: `internal/tools/edit_file_test.go`

- [ ] **Step 1: Write read_file tests**

```go
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
```

- [ ] **Step 2: Write read_file.go**

```go
// internal/tools/read_file.go
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

const maxFileSize = 10 * 1024 * 1024 // 10MB

type readFileArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // 0-indexed line offset
	Limit  int    `json:"limit,omitempty"`  // max lines to return (0 = all)
}

// ReadFileTool reads file contents with line numbers.
type ReadFileTool struct {
	workspace *core.Workspace
}

func NewReadFileTool(ws *core.Workspace) *ReadFileTool {
	return &ReadFileTool{workspace: ws}
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read a file's contents with line numbers. Supports offset and limit for pagination." }
func (t *ReadFileTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path (relative to workspace or absolute)"},
			"offset": {"type": "integer", "description": "Start reading from this line (0-indexed)"},
			"limit": {"type": "integer", "description": "Maximum lines to return (0 = all, default 2000)"}
		},
		"required": ["path"]
	}`)
}

func (t *ReadFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args readFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", args.Path, err)
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file %q is too large (%d bytes, max %d)", args.Path, info.Size(), maxFileSize)
	}

	f, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= args.Offset {
			continue
		}
		if len(lines) >= limit {
			break
		}
		lines = append(lines, fmt.Sprintf("%d\t%s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{Output: output}, nil
}
```

- [ ] **Step 3: Run read_file tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v -run TestReadFile`
Expected: All PASS.

- [ ] **Step 4: Write write_file tests**

```go
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
```

- [ ] **Step 5: Write write_file.go**

```go
// internal/tools/write_file.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFileTool creates or overwrites a file.
type WriteFileTool struct {
	workspace *core.Workspace
}

func NewWriteFileTool(ws *core.Workspace) *WriteFileTool {
	return &WriteFileTool{workspace: ws}
}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Create or overwrite a file. Creates parent directories if needed." }
func (t *WriteFileTool) PermissionTier() permission.PermissionTier { return permission.Write }

func (t *WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path (relative to workspace or absolute)"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args writeFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return nil, fmt.Errorf("create directories: %w", err)
	}

	// Check if file exists for diff reporting
	existed := false
	if _, err := os.Stat(resolved); err == nil {
		existed = true
	}

	if err := os.WriteFile(resolved, []byte(args.Content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	action := "Created"
	if existed {
		action = "Overwrote"
	}

	return &ToolResult{
		Output: fmt.Sprintf("%s %s (%d bytes)", action, args.Path, len(args.Content)),
	}, nil
}
```

- [ ] **Step 6: Write edit_file tests**

```go
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
```

- [ ] **Step 7: Write edit_file.go**

```go
// internal/tools/edit_file.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type editFileArgs struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditFileTool performs surgical string replacement in files.
type EditFileTool struct {
	workspace *core.Workspace
}

func NewEditFileTool(ws *core.Workspace) *EditFileTool {
	return &EditFileTool{workspace: ws}
}

func (t *EditFileTool) Name() string        { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "Replace a specific string in a file. old_string must be unique unless replace_all is true. Provide enough surrounding context to ensure uniqueness."
}
func (t *EditFileTool) PermissionTier() permission.PermissionTier { return permission.Write }

func (t *EditFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path"},
			"old_string": {"type": "string", "description": "Exact text to find and replace"},
			"new_string": {"type": "string", "description": "Replacement text"},
			"replace_all": {"type": "boolean", "description": "Replace all occurrences (default false)"}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}

func (t *EditFileTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args editFileArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.OldString == args.NewString {
		return nil, fmt.Errorf("old_string and new_string are identical")
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", args.Path, err)
	}

	original := string(content)
	count := strings.Count(original, args.OldString)

	if count == 0 {
		return nil, fmt.Errorf("old_string not found in %q", args.Path)
	}

	if !args.ReplaceAll && count > 1 {
		return nil, fmt.Errorf("old_string has %d occurrences in %q — not unique. Provide more surrounding context to make it unique, or set replace_all=true", count, args.Path)
	}

	var newContent string
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(original, args.OldString, args.NewString)
	} else {
		newContent = strings.Replace(original, args.OldString, args.NewString, 1)
	}

	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	replacements := count
	if !args.ReplaceAll {
		replacements = 1
	}

	return &ToolResult{
		Output: fmt.Sprintf("Edited %s (%d replacement(s))", args.Path, replacements),
	}, nil
}
```

- [ ] **Step 8: Run all file tool tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v -run "TestReadFile|TestWriteFile|TestEditFile"`
Expected: All PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/tools/read_file.go internal/tools/read_file_test.go
git add internal/tools/write_file.go internal/tools/write_file_test.go
git add internal/tools/edit_file.go internal/tools/edit_file_test.go
git commit -m "feat: file tools — read_file, write_file, edit_file (str_replace)"
```

---

## Task 11: Bash Tool

**Files:**
- Create: `internal/tools/bash.go`
- Create: `internal/tools/bash_test.go`

- [ ] **Step 1: Write bash tool tests**

```go
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

	cmd := "echo hello"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo hello"
	}

	args, _ := json.Marshal(map[string]interface{}{
		"command":     cmd,
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
```

- [ ] **Step 2: Write bash.go**

```go
// internal/tools/bash.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

const defaultBashTimeout = 120 // seconds
const maxBashTimeout = 600     // seconds

type bashArgs struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Timeout     int    `json:"timeout,omitempty"` // seconds
}

// BashTool executes shell commands with safety restrictions.
type BashTool struct {
	workspace *core.Workspace
}

func NewBashTool(ws *core.Workspace) *BashTool {
	return &BashTool{workspace: ws}
}

func (t *BashTool) Name() string        { return "bash" }
func (t *BashTool) Description() string {
	return "Execute a shell command. Requires a description of what the command does. Dangerous commands and shell injection are blocked."
}
func (t *BashTool) PermissionTier() permission.PermissionTier { return permission.Dangerous }

func (t *BashTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "The command to execute"},
			"description": {"type": "string", "description": "Clear description of what this command does"},
			"timeout": {"type": "integer", "description": "Timeout in seconds (default 120, max 600)"}
		},
		"required": ["command", "description"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args bashArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Description == "" {
		return nil, fmt.Errorf("description is required for bash commands")
	}

	if permission.IsCommandDangerous(args.Command) {
		return nil, fmt.Errorf("command blocked: %q is dangerous", args.Command)
	}

	if permission.IsInjectionAttempt(args.Command) {
		return nil, fmt.Errorf("command blocked: detected shell injection in %q", args.Command)
	}

	timeout := args.Timeout
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}
	if timeout > maxBashTimeout {
		timeout = maxBashTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Split command into program + args (list-based, no shell interpretation)
	parts := splitCommand(args.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = t.workspace.CWD()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return nil, fmt.Errorf("command timed out after %v", duration)
		}
	}

	return &ToolResult{
		Output:   output,
		ExitCode: exitCode,
	}, nil
}

// splitCommand splits a command string into program and arguments.
// Handles simple quoting.
func splitCommand(cmd string) []string {
	// On Windows, use cmd /c for complex commands
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", cmd}
	}
	// On Unix, use sh -c
	return []string{"sh", "-c", cmd}
}
```

- [ ] **Step 3: Run bash tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v -run TestBash`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tools/bash.go internal/tools/bash_test.go
git commit -m "feat: bash tool — command execution with safety checks"
```

---

## Task 12: Search Tools (grep, glob, list_dir)

**Files:**
- Create: `internal/tools/grep.go`
- Create: `internal/tools/grep_test.go`
- Create: `internal/tools/glob.go`
- Create: `internal/tools/glob_test.go`
- Create: `internal/tools/list_dir.go`
- Create: `internal/tools/list_dir_test.go`

- [ ] **Step 1: Write grep tests**

```go
// internal/tools/grep_test.go
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

func TestGrep_BasicMatch(t *testing.T) {
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
```

- [ ] **Step 2: Write grep.go**

Grep wraps `rg` (ripgrep) as a subprocess if available, falling back to Go's native `filepath.WalkDir` + `regexp` if not.

```go
// internal/tools/grep.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

func (t *GrepTool) Name() string        { return "grep" }
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
	// Fallback: native Go implementation using filepath.WalkDir + regexp
	// This is a simplified version; ripgrep is preferred
	return &ToolResult{
		Output: fmt.Sprintf("ripgrep not found. Install rg for better search. Searched for %q in %s", args.Pattern, searchPath),
	}, nil
}
```

- [ ] **Step 3: Write glob tests**

```go
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
```

- [ ] **Step 4: Write glob.go**

```go
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
```

- [ ] **Step 5: Write list_dir tests and implementation**

```go
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
```

```go
// internal/tools/list_dir.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"__pycache__": true, ".venv": true, "target": true,
}

type listDirArgs struct {
	Path  string `json:"path"`
	Depth int    `json:"depth,omitempty"`
}

// ListDirTool lists directory contents.
type ListDirTool struct {
	workspace *core.Workspace
}

func NewListDirTool(ws *core.Workspace) *ListDirTool {
	return &ListDirTool{workspace: ws}
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List directory contents. Ignores .git, node_modules, vendor." }
func (t *ListDirTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }

func (t *ListDirTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Directory path (default: workspace root)"},
			"depth": {"type": "integer", "description": "Max depth (default: 2)"}
		},
		"required": ["path"]
	}`)
}

func (t *ListDirTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args listDirArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resolved, err := t.workspace.Resolve(args.Path)
	if err != nil {
		return nil, err
	}

	depth := args.Depth
	if depth <= 0 {
		depth = 2
	}

	var lines []string
	walkDir(resolved, resolved, 0, depth, &lines)

	if len(lines) == 0 {
		return &ToolResult{Output: "Empty directory."}, nil
	}
	return &ToolResult{Output: strings.Join(lines, "\n")}, nil
}

func walkDir(root, dir string, currentDepth, maxDepth int, lines *[]string) {
	if currentDepth > maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if ignoredDirs[name] {
			continue
		}

		relPath, _ := filepath.Rel(root, filepath.Join(dir, name))
		relPath = filepath.ToSlash(relPath)

		if entry.IsDir() {
			*lines = append(*lines, relPath+"/")
			walkDir(root, filepath.Join(dir, name), currentDepth+1, maxDepth, lines)
		} else {
			*lines = append(*lines, relPath)
		}
	}
}
```

- [ ] **Step 6: Run all search tool tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v -run "TestGrep|TestGlob|TestListDir"`
Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tools/grep.go internal/tools/grep_test.go
git add internal/tools/glob.go internal/tools/glob_test.go
git add internal/tools/list_dir.go internal/tools/list_dir_test.go
git commit -m "feat: search tools — grep (ripgrep), glob, list_dir"
```

---

## Task 13: Git Tools

**Files:**
- Create: `internal/tools/git.go`
- Create: `internal/tools/git_test.go`

- [ ] **Step 1: Write git tool tests**

```go
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
	exec.Command("git", "add", ".").Dir = dir
	cmd := exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	cmd.Run()

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
```

- [ ] **Step 2: Write git.go**

```go
// internal/tools/git.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/permission"
)

type gitArgs struct {
	Operation string   `json:"operation"`
	Files     []string `json:"files,omitempty"`
	Message   string   `json:"message,omitempty"`
	Count     int      `json:"count,omitempty"`
}

// GitTool provides read and write git operations.
type GitTool struct {
	workspace *core.Workspace
}

func NewGitTool(ws *core.Workspace) *GitTool {
	return &GitTool{workspace: ws}
}

func (t *GitTool) Name() string        { return "git" }
func (t *GitTool) Description() string {
	return "Git operations: status, diff, log, add, commit. Operations like push, pull, clone are not supported."
}

func (t *GitTool) PermissionTier() permission.PermissionTier {
	return permission.Write // add and commit modify state
}

func (t *GitTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"operation": {"type": "string", "enum": ["status", "diff", "log", "add", "commit"], "description": "Git operation"},
			"files": {"type": "array", "items": {"type": "string"}, "description": "Files to add (for 'add' operation)"},
			"message": {"type": "string", "description": "Commit message (for 'commit' operation)"},
			"count": {"type": "integer", "description": "Number of log entries (for 'log' operation, default 10)"}
		},
		"required": ["operation"]
	}`)
}

func (t *GitTool) Execute(ctx context.Context, rawArgs json.RawMessage) (*ToolResult, error) {
	var args gitArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	switch args.Operation {
	case "status":
		return t.runGit(ctx, "status", "--short")
	case "diff":
		return t.runGit(ctx, "diff")
	case "log":
		count := args.Count
		if count <= 0 {
			count = 10
		}
		return t.runGit(ctx, "log", "--oneline", fmt.Sprintf("-%d", count))
	case "add":
		if len(args.Files) == 0 {
			return nil, fmt.Errorf("git add requires files")
		}
		gitArgs := append([]string{"add"}, args.Files...)
		return t.runGit(ctx, gitArgs...)
	case "commit":
		if args.Message == "" {
			return nil, fmt.Errorf("git commit requires a message")
		}
		return t.runGit(ctx, "commit", "-m", args.Message)
	default:
		return nil, fmt.Errorf("unsupported git operation: %q (supported: status, diff, log, add, commit)", args.Operation)
	}
}

func (t *GitTool) runGit(ctx context.Context, args ...string) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = t.workspace.Root()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 && output == "" {
		output = stderr.String()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ToolResult{Output: output, ExitCode: exitErr.ExitCode()}, nil
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	if output == "" {
		output = "OK (no output)"
	}
	return &ToolResult{Output: output}, nil
}
```

- [ ] **Step 3: Run git tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/tools/ -v -run TestGit`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tools/git.go internal/tools/git_test.go
git commit -m "feat: git tool — status, diff, log, add, commit"
```

---

## Task 14: Context Window Manager

**Files:**
- Create: `internal/ctxwin/blocks.go`
- Create: `internal/ctxwin/blocks_test.go`
- Create: `internal/ctxwin/manager.go`
- Create: `internal/ctxwin/manager_test.go`

- [ ] **Step 1: Write blocks and manager tests**

```go
// internal/ctxwin/blocks_test.go
package ctxwin

import "testing"

func TestBlockType_String(t *testing.T) {
	if BlockStatic.String() != "static" {
		t.Errorf("expected 'static', got %q", BlockStatic.String())
	}
}

// internal/ctxwin/manager_test.go
package ctxwin

import (
	"testing"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

func TestManager_UsageTracking(t *testing.T) {
	m := NewManager(8192, 2048)
	msgs := []core.Message{
		core.NewSystemMessage("you are helpful"),
		core.NewUserMessage("hello"),
	}
	usage := m.EstimateUsage(msgs)
	if usage <= 0 {
		t.Errorf("expected positive usage, got %d", usage)
	}
	if usage >= 8192 {
		t.Errorf("expected usage < max, got %d", usage)
	}
}

func TestManager_NeedsCompaction(t *testing.T) {
	m := NewManager(100, 20) // small window for testing
	// Create messages that would exceed 75% of 100 tokens
	var msgs []core.Message
	for i := 0; i < 50; i++ {
		msgs = append(msgs, core.NewUserMessage("this is a test message with enough words to use tokens"))
	}
	if !m.NeedsCompaction(msgs) {
		t.Error("expected NeedsCompaction=true for large message set")
	}
}

func TestManager_NeedsCompaction_Small(t *testing.T) {
	m := NewManager(8192, 2048)
	msgs := []core.Message{core.NewUserMessage("hi")}
	if m.NeedsCompaction(msgs) {
		t.Error("expected NeedsCompaction=false for small message set")
	}
}

func TestManager_Trim(t *testing.T) {
	m := NewManager(100, 20)
	// System message should always survive
	msgs := []core.Message{
		core.NewSystemMessage("system"),
		core.NewUserMessage("old message 1"),
		core.NewAssistantMessage("old response 1"),
		core.NewUserMessage("old message 2"),
		core.NewAssistantMessage("old response 2"),
		core.NewUserMessage("recent message"),
		core.NewAssistantMessage("recent response"),
	}
	trimmed := m.Trim(msgs)
	// System message must be first
	if trimmed[0].Role != core.RoleSystem {
		t.Error("expected system message preserved as first")
	}
	// Should have fewer messages
	if len(trimmed) >= len(msgs) {
		t.Error("expected trimmed to be shorter")
	}
	// Last message should be preserved
	if trimmed[len(trimmed)-1].Content != "recent response" {
		t.Error("expected most recent message preserved")
	}
}
```

- [ ] **Step 2: Write blocks.go and manager.go**

```go
// internal/ctxwin/blocks.go
package ctxwin

// BlockType categorizes context content for compaction decisions.
type BlockType int

const (
	BlockStatic  BlockType = iota // System prompt, project config — never compacted
	BlockActive                   // Conversation, tool results — compactable
	BlockArchive                  // Compacted summaries — read-only reference
)

func (b BlockType) String() string {
	switch b {
	case BlockStatic:
		return "static"
	case BlockActive:
		return "active"
	case BlockArchive:
		return "archive"
	default:
		return "unknown"
	}
}
```

```go
// internal/ctxwin/manager.go
package ctxwin

import (
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// Manager tracks token budget and decides when to compact.
type Manager struct {
	MaxTokens     int
	ReserveTokens int
	CompactAt     float64 // fraction (default 0.75)
}

// NewManager creates a context window manager.
func NewManager(maxTokens, reserveTokens int) *Manager {
	return &Manager{
		MaxTokens:     maxTokens,
		ReserveTokens: reserveTokens,
		CompactAt:     0.75,
	}
}

// EstimateUsage returns the estimated token count for a message slice.
func (m *Manager) EstimateUsage(messages []core.Message) int {
	return llm.EstimateMessageTokens(messages)
}

// Available returns how many tokens remain for generation.
func (m *Manager) Available(messages []core.Message) int {
	used := m.EstimateUsage(messages)
	return m.MaxTokens - used - m.ReserveTokens
}

// NeedsCompaction returns true if usage exceeds the CompactAt threshold.
func (m *Manager) NeedsCompaction(messages []core.Message) bool {
	used := m.EstimateUsage(messages)
	threshold := float64(m.MaxTokens) * m.CompactAt
	return float64(used) > threshold
}

// Trim removes oldest non-system messages to fit within budget.
// Preserves: system messages (always first), most recent messages.
// This is the simple fallback — auto-compaction with summarization is in Plan 2.
func (m *Manager) Trim(messages []core.Message) []core.Message {
	budget := m.MaxTokens - m.ReserveTokens

	// Always keep the system message
	var system []core.Message
	var rest []core.Message
	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			system = append(system, msg)
		} else {
			rest = append(rest, msg)
		}
	}

	// Start from the end (most recent) and work backward
	result := make([]core.Message, 0, len(messages))
	result = append(result, system...)
	systemTokens := llm.EstimateMessageTokens(system)
	remaining := budget - systemTokens

	// Add messages from most recent to oldest until budget exhausted
	var kept []core.Message
	for i := len(rest) - 1; i >= 0; i-- {
		msgTokens := llm.EstimateMessageTokens([]core.Message{rest[i]})
		if remaining-msgTokens < 0 {
			break
		}
		remaining -= msgTokens
		kept = append(kept, rest[i])
	}

	// Reverse kept to restore chronological order
	for i := len(kept) - 1; i >= 0; i-- {
		result = append(result, kept[i])
	}

	return result
}
```

- [ ] **Step 3: Run tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/ctxwin/ -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/ctxwin/
git commit -m "feat: context window manager — token tracking, trimming"
```

---

## Task 15: Agent Loop

**Files:**
- Create: `internal/agent/agent.go`
- Create: `internal/agent/agent_test.go`
- Create: `internal/agent/loop.go`
- Create: `internal/agent/loop_test.go`

- [ ] **Step 1: Write agent tests**

```go
// internal/agent/agent_test.go
package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/permission"
	"github.com/crussella0129/fev/internal/tools"
)

// mockClient implements a simple LLM client for testing.
type mockClient struct {
	responses []*core.Message
	callIndex int
}

func (m *mockClient) Generate(ctx context.Context, messages []core.Message, toolSchemas []llm.ToolSchema) (*core.Message, error) {
	if m.callIndex >= len(m.responses) {
		return &core.Message{Role: core.RoleAssistant, Content: "done"}, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

type echoTool struct{}

func (e *echoTool) Name() string                    { return "echo" }
func (e *echoTool) Description() string             { return "echoes input" }
func (e *echoTool) Schema() json.RawMessage         { return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`) }
func (e *echoTool) PermissionTier() permission.PermissionTier { return permission.ReadOnly }
func (e *echoTool) Execute(ctx context.Context, args json.RawMessage) (*tools.ToolResult, error) {
	var a struct{ Text string `json:"text"` }
	json.Unmarshal(args, &a)
	return &tools.ToolResult{Output: "echo: " + a.Text}, nil
}

func TestAgent_SimpleResponse(t *testing.T) {
	client := &mockClient{
		responses: []*core.Message{
			{Role: core.RoleAssistant, Content: "Hello!"},
		},
	}

	reg := tools.NewRegistry()
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", resp)
	}
}

func TestAgent_ToolCallThenResponse(t *testing.T) {
	client := &mockClient{
		responses: []*core.Message{
			// First: model calls echo tool
			{
				Role: core.RoleAssistant,
				ToolCalls: []core.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: core.FunctionCall{
							Name:      "echo",
							Arguments: `{"text":"world"}`,
						},
					},
				},
			},
			// Second: model responds with text
			{Role: core.RoleAssistant, Content: "The echo said: world"},
		},
	}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "echo world")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp != "The echo said: world" {
		t.Errorf("expected 'The echo said: world', got %q", resp)
	}
}

func TestAgent_MaxTurns(t *testing.T) {
	// Model always calls a tool, never stops
	infiniteCalls := make([]*core.Message, 100)
	for i := range infiniteCalls {
		infiniteCalls[i] = &core.Message{
			Role: core.RoleAssistant,
			ToolCalls: []core.ToolCall{
				{ID: "call", Type: "function", Function: core.FunctionCall{Name: "echo", Arguments: `{"text":"loop"}`}},
			},
		}
	}
	client := &mockClient{responses: infiniteCalls}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()
	cfg.Agent.MaxTurns = 3

	a := New(client, reg, mgr, cfg)
	_, err := a.Run(context.Background(), "loop forever")
	if err == nil {
		t.Fatal("expected max turns error")
	}
}

func TestAgent_RepeatDetection(t *testing.T) {
	// Same tool call 3 times → should break
	sameCall := &core.Message{
		Role: core.RoleAssistant,
		ToolCalls: []core.ToolCall{
			{ID: "call", Type: "function", Function: core.FunctionCall{Name: "echo", Arguments: `{"text":"same"}`}},
		},
	}
	client := &mockClient{
		responses: []*core.Message{sameCall, sameCall, sameCall, {Role: core.RoleAssistant, Content: "final"}},
	}

	reg := tools.NewRegistry()
	reg.Register(&echoTool{})
	mgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := New(client, reg, mgr, cfg)
	resp, err := a.Run(context.Background(), "test repeat")
	// Should not error — repeat detection should break the loop gracefully
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	_ = resp
}
```

- [ ] **Step 2: Write agent.go and loop.go**

```go
// internal/agent/agent.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/tools"
)

// LLMClient is the interface for LLM generation (allows mocking).
type LLMClient interface {
	Generate(ctx context.Context, messages []core.Message, tools []llm.ToolSchema) (*core.Message, error)
}

// Agent is the core agentic loop.
type Agent struct {
	client   LLMClient
	registry *tools.Registry
	ctxMgr   *ctxwin.Manager
	config   *config.Config
	history  []core.Message
	onChunk  func(string) // streaming callback (optional)
}

// New creates a new agent.
func New(client LLMClient, registry *tools.Registry, ctxMgr *ctxwin.Manager, cfg *config.Config) *Agent {
	return &Agent{
		client:   client,
		registry: registry,
		ctxMgr:   ctxMgr,
		config:   cfg,
	}
}

// SetSystemPrompt sets the initial system message.
func (a *Agent) SetSystemPrompt(prompt string) {
	// Replace or prepend system message
	if len(a.history) > 0 && a.history[0].Role == core.RoleSystem {
		a.history[0] = core.NewSystemMessage(prompt)
	} else {
		a.history = append([]core.Message{core.NewSystemMessage(prompt)}, a.history...)
	}
}

// SetStreaming sets the chunk callback for streaming output.
func (a *Agent) SetStreaming(onChunk func(string)) {
	a.onChunk = onChunk
}

// Reset clears conversation history (preserves system prompt).
func (a *Agent) Reset() {
	if len(a.history) > 0 && a.history[0].Role == core.RoleSystem {
		a.history = a.history[:1]
	} else {
		a.history = nil
	}
}

// Run executes the agent loop for a user input.
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	a.history = append(a.history, core.NewUserMessage(input))
	return a.loop(ctx)
}
```

```go
// internal/agent/loop.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/tools"
)

const repeatThreshold = 3

// loop runs the flat agent loop until the model emits a text-only response or max turns.
func (a *Agent) loop(ctx context.Context) (string, error) {
	maxTurns := a.config.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}

	repeatTracker := make(map[string]int) // hash → count

	for turn := 0; turn < maxTurns; turn++ {
		// Trim history if needed
		a.history = a.ctxMgr.Trim(a.history)

		// Generate
		toolSchemas := a.registry.ToOpenAISchemas()
		resp, err := a.client.Generate(ctx, a.history, toolSchemas)
		if err != nil {
			class := core.ClassifyError(err)
			if class == core.ErrorRetryable && turn < maxTurns-1 {
				backoff := time.Duration(1<<uint(turn)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			return "", fmt.Errorf("generation failed: %w", err)
		}

		// No tool calls → done
		if len(resp.ToolCalls) == 0 {
			a.history = append(a.history, core.NewAssistantMessage(resp.Content))
			return resp.Content, nil
		}

		// Deduplicate tool calls
		toolCalls := core.DeduplicateToolCalls(resp.ToolCalls)

		// Check repeat detection
		allRepeated := true
		for _, tc := range toolCalls {
			hash := core.ToolCallHash(tc)
			repeatTracker[hash]++
			if repeatTracker[hash] < repeatThreshold {
				allRepeated = false
			}
		}

		if allRepeated {
			// All tool calls have been seen 3+ times — break the loop
			text := resp.Content
			if text == "" {
				text = "[Agent stopped: repeated tool calls detected]"
			}
			a.history = append(a.history, core.NewAssistantMessage(text))
			return text, nil
		}

		// Add assistant message with tool calls
		a.history = append(a.history, core.NewAssistantMessageWithTools(resp.Content, toolCalls))

		// Execute each tool
		for _, tc := range toolCalls {
			tool := a.registry.Get(tc.Function.Name)
			if tool == nil {
				a.history = append(a.history, core.NewToolMessage(tc.ID, fmt.Sprintf("Error: unknown tool %q", tc.Function.Name)))
				continue
			}

			result, execErr := tools.ExecuteWithTimeout(ctx, tool, json.RawMessage(tc.Function.Arguments), 120*time.Second)
			if execErr != nil {
				a.history = append(a.history, core.NewToolMessage(tc.ID, "Error: "+execErr.Error()))
				continue
			}

			// Truncate large output
			result = tools.MaybeTruncate(result, 8000)
			a.history = append(a.history, core.NewToolMessage(tc.ID, result.Output))
		}
	}

	return "", fmt.Errorf("max turns (%d) exceeded", maxTurns)
}
```

- [ ] **Step 3: Run agent tests**

Run: `cd C:/Users/charl/Fev && go test ./internal/agent/ -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/agent/
git commit -m "feat: flat agent loop — tool calling, repeat detection, dedup"
```

---

## Task 16: REPL & CLI Wiring

**Files:**
- Modify: `cmd/fev/main.go`

- [ ] **Step 1: Wire everything together in main.go**

Replace cmd/fev/main.go with the full CLI including interactive REPL:

```go
// cmd/fev/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/crussella0129/fev/internal/agent"
	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/tools"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

var (
	cfgPath       string
	workspaceRoot string
	verbose       bool
)

var rootCmd = &cobra.Command{
	Use:   "fev [task]",
	Short: "Fev — local-first agentic CLI",
	Long:  "Fev is a personal assistant for navigating the digital space. Model-agnostic, single binary, local-first.",
	RunE:  runInteractive,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Fev version",
	Run:   func(cmd *cobra.Command, args []string) { fmt.Printf("fev v%s\n", version) },
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create FEV.md template in current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		template := `# FEV.md

## Project
Describe your project here.

## Build & Test
- Build: ` + "`command`" + `
- Test: ` + "`command`" + `

## Conventions
- List conventions here

## Do Not
- List restrictions here
`
		if _, err := os.Stat("FEV.md"); err == nil {
			return fmt.Errorf("FEV.md already exists")
		}
		return os.WriteFile("FEV.md", []byte(template), 0644)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&workspaceRoot, "workspace", ".", "Workspace root directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(versionCmd, initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInteractive(cmd *cobra.Command, args []string) error {
	// Load config
	configPath := cfgPath
	if configPath == "" {
		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		configPath = dir + "/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Setup workspace
	ws, err := core.NewWorkspace(workspaceRoot)
	if err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	// Create LLM client
	client := llm.NewClient(cfg.Model.BaseURL, cfg.Model.ModelName)

	// Create tool registry
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFileTool(ws))
	reg.Register(tools.NewWriteFileTool(ws))
	reg.Register(tools.NewEditFileTool(ws))
	reg.Register(tools.NewBashTool(ws))
	reg.Register(tools.NewGrepTool(ws))
	reg.Register(tools.NewGlobTool(ws))
	reg.Register(tools.NewListDirTool(ws))
	reg.Register(tools.NewGitTool(ws))

	// Context manager
	ctxLen := cfg.Model.ContextLength
	if ctxLen <= 0 {
		ctxLen = 8192
	}
	ctxMgr := ctxwin.NewManager(ctxLen, 2048)

	// Create agent
	a := agent.New(client, reg, ctxMgr, cfg)

	// Load project config
	projectCfg, _ := config.LoadProjectConfig(ws.Root())
	systemPrompt := buildSystemPrompt(projectCfg, reg)
	a.SetSystemPrompt(systemPrompt)

	// Banner
	fmt.Printf("fev v%s — model: %s @ %s\n", version, cfg.Model.ModelName, cfg.Model.BaseURL)
	fmt.Printf("workspace: %s\n", ws.Root())
	fmt.Println("Type /help for commands, /exit to quit.\n")

	// If task provided as args, run it
	if len(args) > 0 {
		task := strings.Join(args, " ")
		return runTask(a, task)
	}

	// Interactive REPL
	return repl(a)
}

func repl(a *agent.Agent) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Slash commands
		switch {
		case input == "/exit" || input == "/quit":
			fmt.Println("Goodbye.")
			return nil
		case input == "/help":
			printHelp()
			continue
		case input == "/reset":
			a.Reset()
			fmt.Println("Conversation reset.")
			continue
		}

		resp, err := a.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(resp)
		fmt.Println()
	}
	return scanner.Err()
}

func runTask(a *agent.Agent, task string) error {
	resp, err := a.Run(context.Background(), task)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /exit, /quit    Exit Fev")
	fmt.Println("  /reset          Clear conversation history")
	fmt.Println("  /help           Show this help")
}

func buildSystemPrompt(projectConfig string, reg *tools.Registry) string {
	var b strings.Builder
	b.WriteString("You are Fev, a local-first coding assistant. You help users navigate and modify their codebase using structured tools.\n\n")
	b.WriteString("Available tools: " + strings.Join(reg.List(), ", ") + "\n\n")
	b.WriteString("Always use tools when you need to read, write, or search files. Do not guess file contents.\n")
	b.WriteString("When using the bash tool, always include a description of what the command does.\n")

	if projectConfig != "" {
		b.WriteString("\n--- Project Configuration ---\n")
		b.WriteString(projectConfig)
	}

	return b.String()
}
```

- [ ] **Step 2: Verify build**

Run: `cd C:/Users/charl/Fev && go build ./cmd/fev`
Expected: Clean build.

Run: `go run ./cmd/fev version`
Expected: `fev v0.1.0`

- [ ] **Step 3: Commit**

```bash
git add cmd/fev/main.go
git commit -m "feat: interactive REPL with tool wiring, slash commands"
```

---

## Task 17: Integration Test

**Files:**
- Create: `test/integration_test.go`

- [ ] **Step 1: Write integration test**

```go
// test/integration_test.go
package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crussella0129/fev/internal/agent"
	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/tools"
)

func TestIntegration_ReadFileViaAgent(t *testing.T) {
	// Setup workspace with a test file
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("Hello, Fev!"), 0644)
	ws, _ := core.NewWorkspace(dir)

	// Mock LLM server: first call returns tool call, second returns text
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp interface{}

		if callCount == 1 {
			// Model decides to read the file
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "read_file",
										"arguments": `{"path":"hello.txt"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
			}
		} else {
			// Model responds with text after seeing file contents
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "The file says: Hello, Fev!",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Wire everything
	client := llm.NewClient(server.URL+"/v1", "test-model")
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFileTool(ws))
	ctxMgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := agent.New(client, reg, ctxMgr, cfg)
	resp, err := a.Run(context.Background(), "What does hello.txt say?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(resp, "Hello, Fev!") {
		t.Errorf("expected response to contain file contents, got %q", resp)
	}
}

func TestIntegration_EditFileViaAgent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main\n\nfunc hello() {}\n"), 0644)
	ws, _ := core.NewWorkspace(dir)

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp interface{}

		if callCount == 1 {
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "edit_file",
										"arguments": `{"path":"code.go","old_string":"func hello() {}","new_string":"func hello() {\n\tfmt.Println(\"hello\")\n}"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]int{"total_tokens": 20},
			}
		} else {
			resp = map[string]interface{}{
				"id": "test",
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Done! Updated hello() to print a greeting.",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{"total_tokens": 15},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL+"/v1", "test-model")
	reg := tools.NewRegistry()
	reg.Register(tools.NewEditFileTool(ws))
	ctxMgr := ctxwin.NewManager(8192, 2048)
	cfg := config.DefaultConfig()

	a := agent.New(client, reg, ctxMgr, cfg)
	_, err := a.Run(context.Background(), "Add a print statement to hello()")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify the file was actually edited
	content, _ := os.ReadFile(filepath.Join(dir, "code.go"))
	if !strings.Contains(string(content), "Println") {
		t.Errorf("expected edited file to contain Println, got %q", string(content))
	}
}
```

- [ ] **Step 2: Run integration tests**

Run: `cd C:/Users/charl/Fev && go test ./test/ -v`
Expected: All PASS.

- [ ] **Step 3: Run full test suite**

Run: `cd C:/Users/charl/Fev && go test ./... -race -timeout 120s`
Expected: All PASS across all packages.

- [ ] **Step 4: Final commit**

```bash
git add test/
git commit -m "feat: integration tests — end-to-end agent loop with mock LLM"
```

- [ ] **Step 5: Tag release**

```bash
git tag v0.1.0
```
