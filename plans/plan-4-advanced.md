# Plan 4: Advanced Features — Subagents, Plan Mode, MCP

> **Prerequisite:** Plan 1 (Foundation) must be complete. Plans 2 and 3 are recommended but not strictly required.
> **Goal:** Add parallel subagent execution, user-approval plan mode, and MCP integration.
> **Estimated effort:** 6 tasks, ~45 TDD steps

---

## What This Plan Builds

1. **Subagents** — Fev can spawn parallel worker goroutines for read-only tasks (research, search, analysis), collect their results, and synthesize them
2. **Plan mode** — `/plan` command shows a proposed plan before execution, user approves or rejects
3. **MCP bridge** — Fev connects to external MCP (Model Context Protocol) servers, discovers their tools, and registers them alongside native tools

---

## Go Concepts You'll Need

### Goroutines + Channels + Select (the core of Go concurrency)

This is Go's killer feature. A goroutine is a lightweight thread (you can have thousands). A channel is a typed pipe between goroutines. `select` lets you wait on multiple channels at once.

```go
// Spawn 3 workers, collect results with timeout
func searchInParallel(queries []string) []string {
    results := make(chan string, len(queries))  // buffered channel
    
    for _, q := range queries {
        go func(query string) {
            result := doSearch(query)
            results <- result  // send result into channel
        }(q)
    }
    
    // Collect results with timeout
    var collected []string
    timeout := time.After(30 * time.Second)
    for i := 0; i < len(queries); i++ {
        select {
        case r := <-results:
            collected = append(collected, r)
        case <-timeout:
            return collected  // return what we have
        }
    }
    return collected
}
```

### Context for cancellation

```go
// Parent creates a cancellable context
ctx, cancel := context.WithCancel(parentCtx)
defer cancel()  // always cancel to free resources

go func() {
    // Child checks ctx.Done() to know when to stop
    select {
    case <-ctx.Done():
        return  // parent cancelled us
    case result := <-doWork():
        // continue working
    }
}()

// Parent can cancel all children at once
cancel()
```

### JSON-RPC (for MCP)

MCP uses JSON-RPC 2.0 over stdin/stdout. Here's the format:

```json
// Request
{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}

// Response  
{"jsonrpc": "2.0", "id": 1, "result": {"tools": [...]}}
```

In Go:
```go
type jsonRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *jsonRPCError   `json:"error,omitempty"`
}
```

### os/exec for subprocess management

MCP servers run as subprocesses. You need to control their stdin/stdout:

```go
cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem")
cmd.Stdin = stdinPipe   // you write JSON-RPC requests here
cmd.Stdout = stdoutPipe // you read JSON-RPC responses here
cmd.Stderr = os.Stderr  // let errors go to terminal
cmd.Start()             // start but don't wait
// ... communicate via pipes ...
cmd.Process.Kill()      // when done
```

---

## File Structure

```
internal/
  agent/
    subagent.go          # Subagent spawning and result collection
    subagent_test.go
    planmode.go          # /plan command implementation
    planmode_test.go
  mcp/
    client.go            # JSON-RPC client over stdin/stdout
    client_test.go
    discovery.go          # Find and parse .mcp.json configs
    discovery_test.go
    bridge.go            # Register MCP tools into Fev's tool registry
    bridge_test.go
    types.go             # MCP protocol types
```

---

## Task 1: Subagent Core

**What you're building:** The ability for the parent agent to spawn read-only worker goroutines.

**Files:** `internal/agent/subagent.go`, `internal/agent/subagent_test.go`

### Design

A subagent is:
- A goroutine running its own mini agent loop
- With a **copy** of the parent's file cache (not a reference — isolation)
- With a **restricted** tool registry (read-only tools only)
- With its own context window and token budget
- With a cancellable context and timeout

The parent decides when to spawn subagents. Each subagent gets a task (a string describing what to research/find), runs the agent loop, and sends its result back via a channel.

### What to implement

```go
package agent

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/crussella0129/fev/internal/config"
    "github.com/crussella0129/fev/internal/core"
    "github.com/crussella0129/fev/internal/ctxwin"
    "github.com/crussella0129/fev/internal/permission"
    "github.com/crussella0129/fev/internal/tools"
)

// SubAgentResult is what a subagent sends back to its parent.
type SubAgentResult struct {
    ID      string
    Task    string        // what it was asked to do
    Output  string        // the result
    Err     error         // nil if successful
    Elapsed time.Duration // how long it took
}

// SubAgentConfig controls subagent behavior.
type SubAgentConfig struct {
    Timeout   time.Duration // max time per subagent (default 60s)
    MaxTokens int           // context budget per subagent
}

// filterReadOnly creates a new registry with only read-only tools.
func filterReadOnly(source *tools.Registry) *tools.Registry {
    filtered := tools.NewRegistry()
    for _, name := range source.List() {
        tool := source.Get(name)
        if tool.PermissionTier() == permission.ReadOnly {
            filtered.Register(tool)
        }
    }
    return filtered
}

// SpawnSubAgents runs tasks in parallel goroutines and collects results.
//
// How it works:
// 1. Creates a read-only tool registry (no write_file, edit_file, bash, git add/commit)
// 2. For each task, spawns a goroutine with its own agent loop
// 3. Each goroutine sends its result to a shared channel
// 4. Parent collects results with a timeout
// 5. If timeout fires, returns whatever results arrived
//
// IMPORTANT: Subagents cannot spawn sub-subagents (single depth).
func SpawnSubAgents(
    ctx context.Context,
    client LLMClient, 
    registry *tools.Registry,
    cfg *config.Config,
    tasks []string,
    subCfg SubAgentConfig,
) []SubAgentResult {
    if subCfg.Timeout == 0 {
        subCfg.Timeout = 60 * time.Second
    }
    if subCfg.MaxTokens == 0 {
        subCfg.MaxTokens = 4096
    }
    
    readOnlyReg := filterReadOnly(registry)
    results := make(chan SubAgentResult, len(tasks))
    
    var wg sync.WaitGroup
    for i, task := range tasks {
        wg.Add(1)
        go func(id int, taskDesc string) {
            defer wg.Done()
            
            subCtx, cancel := context.WithTimeout(ctx, subCfg.Timeout)
            defer cancel()
            
            start := time.Now()
            
            // Create a mini agent with read-only tools
            subMgr := ctxwin.NewManager(subCfg.MaxTokens, 512)
            subAgent := New(client, readOnlyReg, subMgr, cfg)
            subAgent.SetSystemPrompt(
                "You are a research assistant. Answer the question using the available tools. " +
                "Be concise and factual. You can only read files and search — you cannot modify anything.")
            
            output, err := subAgent.Run(subCtx, taskDesc)
            
            results <- SubAgentResult{
                ID:      fmt.Sprintf("sub-%d", id),
                Task:    taskDesc,
                Output:  output,
                Err:     err,
                Elapsed: time.Since(start),
            }
        }(i, task)
    }
    
    // Close channel when all goroutines finish
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results (channel will close when all done, or we hit parent timeout)
    var collected []SubAgentResult
    for r := range results {
        collected = append(collected, r)
    }
    return collected
}
```

### Key design decisions explained

**Why copy the tool registry instead of sharing it?**
Each subagent gets its own `filterReadOnly(registry)`. This is a new Registry object, not a reference. This means:
- Subagents can't register new tools that affect the parent
- The parent can modify its registry without affecting running subagents
- No mutex contention between parent and subagent tool lookups

**Why `sync.WaitGroup` + channel?**
- `WaitGroup` tracks when all goroutines finish (so we can close the channel)
- Channel delivers results as they arrive (not all at once at the end)
- `for r := range results` blocks until the channel is closed

**Why not allow sub-subagents?**
Simplicity. Unbounded recursion leads to goroutine explosions and token waste. The parent can always spawn more subagents if needed.

### Tests

```go
func TestSpawnSubAgents_BasicExecution(t *testing.T) {
    // Mock client that answers questions
    client := &mockClient{
        responses: []*core.Message{
            {Role: core.RoleAssistant, Content: "The answer is 42"},
        },
    }
    
    reg := tools.NewRegistry()
    // Register a read-only mock tool
    reg.Register(&mockReadOnlyTool{})
    
    cfg := config.DefaultConfig()
    tasks := []string{"What is the meaning of life?"}
    
    results := SpawnSubAgents(
        context.Background(), client, reg, cfg, tasks,
        SubAgentConfig{Timeout: 5 * time.Second},
    )
    
    if len(results) != 1 {
        t.Fatalf("expected 1 result, got %d", len(results))
    }
    if results[0].Err != nil {
        t.Fatalf("unexpected error: %v", results[0].Err)
    }
    if results[0].Output == "" {
        t.Error("expected non-empty output")
    }
}

func TestSpawnSubAgents_ParallelExecution(t *testing.T) {
    // Verify multiple tasks run concurrently
    client := &mockClient{...}
    tasks := []string{"task 1", "task 2", "task 3"}
    
    start := time.Now()
    results := SpawnSubAgents(ctx, client, reg, cfg, tasks, 
        SubAgentConfig{Timeout: 5 * time.Second})
    elapsed := time.Since(start)
    
    // If they ran in parallel, total time should be close to 1x, not 3x
    if len(results) != 3 {
        t.Fatalf("expected 3 results, got %d", len(results))
    }
}

func TestSpawnSubAgents_ReadOnlyRestriction(t *testing.T) {
    // Verify subagents can't access write tools
    reg := tools.NewRegistry()
    reg.Register(&mockReadOnlyTool{name: "read_file"})
    reg.Register(&mockWriteTool{name: "write_file"})  // this should be filtered out
    
    // The subagent should only see read_file, not write_file
    // Test by checking the tool schemas sent to the mock client
}

func TestSpawnSubAgents_Timeout(t *testing.T) {
    // Mock client that hangs forever
    client := &hangingClient{}
    results := SpawnSubAgents(ctx, client, reg, cfg, 
        []string{"hang"}, SubAgentConfig{Timeout: 100 * time.Millisecond})
    
    // Should return with error, not hang
    if len(results) == 0 || results[0].Err == nil {
        t.Error("expected timeout error")
    }
}
```

### Commit
```bash
git commit -m "feat: subagent system — goroutine-spawned parallel agents with read-only tools"
```

---

## Task 2: Background Subagent Execution

**What you're building:** The ability to run a subagent in the background while the parent continues its loop.

**Files:** Extend `internal/agent/subagent.go` and tests

### What to implement

```go
// BackgroundAgent runs a subagent in the background.
// The parent can check results later.
type BackgroundAgent struct {
    mu       sync.Mutex
    id       string
    task     string
    done     bool
    result   *SubAgentResult
    cancel   context.CancelFunc
}

// SpawnBackground launches a subagent that runs independently.
// Returns a handle the parent can poll for results.
func SpawnBackground(
    ctx context.Context,
    client LLMClient,
    registry *tools.Registry,
    cfg *config.Config,
    task string,
    subCfg SubAgentConfig,
) *BackgroundAgent {
    bgCtx, cancel := context.WithTimeout(ctx, subCfg.Timeout)
    bg := &BackgroundAgent{
        id:     fmt.Sprintf("bg-%d", time.Now().UnixNano()),
        task:   task,
        cancel: cancel,
    }
    
    go func() {
        defer cancel()
        results := SpawnSubAgents(bgCtx, client, registry, cfg, 
            []string{task}, subCfg)
        
        bg.mu.Lock()
        defer bg.mu.Unlock()
        bg.done = true
        if len(results) > 0 {
            bg.result = &results[0]
        }
    }()
    
    return bg
}

// IsDone checks if the background agent has finished.
func (bg *BackgroundAgent) IsDone() bool {
    bg.mu.Lock()
    defer bg.mu.Unlock()
    return bg.done
}

// Result returns the result, or nil if not done yet.
func (bg *BackgroundAgent) Result() *SubAgentResult {
    bg.mu.Lock()
    defer bg.mu.Unlock()
    return bg.result
}

// Cancel stops the background agent.
func (bg *BackgroundAgent) Cancel() {
    bg.cancel()
}
```

### Tests
- SpawnBackground returns immediately (doesn't block)
- IsDone eventually returns true
- Result contains output after completion
- Cancel stops a running background agent

### Commit
```bash
git commit -m "feat: background subagent execution with polling"
```

---

## Task 3: Plan Mode

**What you're building:** The `/plan` command — LLM proposes a plan, user reviews and approves before execution.

**Files:** `internal/agent/planmode.go`, `internal/agent/planmode_test.go`

### How it works

1. User types `/plan` then their request
2. Fev sends the request to the LLM with a special system prompt: "Generate a plan for this task. List the steps you would take, but do NOT execute any tools. Just describe what you would do."
3. LLM responds with a text plan (no tool calls)
4. Fev displays the plan to the user
5. User types `y` (approve), `n` (reject), or `e` (edit/revise)
6. If approved: Fev executes the plan using the normal agent loop
7. If rejected: return to normal prompt

### What to implement

```go
package agent

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/crussella0129/fev/internal/core"
    "github.com/crussella0129/fev/internal/llm"
)

// PlanResult holds the plan and its approval status.
type PlanResult struct {
    Plan     string // the LLM-generated plan text
    Approved bool   // true if user approved
    Revised  string // if user edited the plan
}

// GeneratePlan asks the LLM to create a plan without executing tools.
func (a *Agent) GeneratePlan(ctx context.Context, request string) (string, error) {
    // Build a planning-specific system prompt
    planPrompt := core.NewSystemMessage(
        "You are a planning assistant. The user will describe a task. " +
        "Generate a step-by-step plan for how you would accomplish it. " +
        "List each step as a numbered item. Describe what tools you would use " +
        "and what files you would read or modify. " +
        "DO NOT execute any tools. Just describe the plan.")
    
    messages := []core.Message{
        planPrompt,
        core.NewUserMessage(request),
    }
    
    // Call LLM with NO tools — force text-only response
    resp, err := a.client.Generate(ctx, messages, nil)
    if err != nil {
        return "", fmt.Errorf("plan generation failed: %w", err)
    }
    
    return resp.Content, nil
}

// ExecutePlan runs a plan through the normal agent loop.
// The plan text is prepended to the user's original request as context.
func (a *Agent) ExecutePlan(ctx context.Context, plan string, originalRequest string) (string, error) {
    prompt := fmt.Sprintf(
        "Execute this plan step by step:\n\n%s\n\nOriginal request: %s\n\n"+
        "Work through each step. Use tools as needed. Report progress.",
        plan, originalRequest)
    
    return a.Run(ctx, prompt)
}

// IsPlanCommand checks if input starts with /plan.
func IsPlanCommand(input string) (bool, string) {
    if strings.HasPrefix(input, "/plan ") {
        return true, strings.TrimPrefix(input, "/plan ")
    }
    if input == "/plan" {
        return true, ""
    }
    return false, ""
}
```

### Tests
- GeneratePlan returns non-empty text
- GeneratePlan sends NO tool schemas (force text-only)
- IsPlanCommand("/plan build a server") returns true, "build a server"
- IsPlanCommand("/plan") returns true, ""
- IsPlanCommand("hello") returns false, ""
- ExecutePlan includes plan in the prompt

### Integration into main.go

In the REPL loop, add:
```go
case strings.HasPrefix(input, "/plan"):
    isPlan, request := agent.IsPlanCommand(input)
    if isPlan {
        if request == "" {
            fmt.Println("Usage: /plan <description of what you want to do>")
            continue
        }
        plan, err := a.GeneratePlan(ctx, request)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            continue
        }
        fmt.Println("\n=== Proposed Plan ===")
        fmt.Println(plan)
        fmt.Println("\nApprove? (y/n): ")
        scanner.Scan()
        if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
            result, err := a.ExecutePlan(ctx, plan, request)
            // handle result...
        }
    }
```

### Commit
```bash
git commit -m "feat: /plan command — propose before executing"
```

---

## Task 4: MCP Types and JSON-RPC Client

**What you're building:** The protocol layer for communicating with MCP servers.

**Files:** `internal/mcp/types.go`, `internal/mcp/client.go`, `internal/mcp/client_test.go`

### What MCP is

**Model Context Protocol (MCP)** is an open standard (from Anthropic) for connecting AI assistants to external tools. An MCP server is a subprocess that speaks JSON-RPC over stdin/stdout. It exposes tools that the AI can call.

For example, `.mcp.json` might define a filesystem server:
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"],
      "env": {}
    }
  }
}
```

Fev launches this process, sends it `tools/list` to discover what tools it has, then calls those tools as needed.

### What to implement

```go
// types.go
package mcp

import "encoding/json"

// ServerConfig defines an MCP server from .mcp.json.
type ServerConfig struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env,omitempty"`
}

// MCPConfig is the top-level .mcp.json structure.
type MCPConfig struct {
    Servers map[string]ServerConfig `json:"mcpServers"`
}

// MCPTool is a tool discovered from an MCP server.
type MCPTool struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    InputSchema json.RawMessage `json:"inputSchema"`
}

// MCPToolResult is the result of calling an MCP tool.
type MCPToolResult struct {
    Content []MCPContent `json:"content"`
    IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
    Type string `json:"type"` // "text" or "image" or "resource"
    Text string `json:"text,omitempty"`
}

// JSON-RPC types
type Request struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params,omitempty"`
}

type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

```go
// client.go
package mcp

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "os/exec"
    "sync"
)

// Client communicates with an MCP server over stdin/stdout.
type Client struct {
    mu      sync.Mutex
    cmd     *exec.Cmd
    stdin   io.WriteCloser
    stdout  *bufio.Scanner
    nextID  int
    name    string  // server name from config
}

// NewClient starts an MCP server subprocess and returns a client.
func NewClient(name string, cfg ServerConfig) (*Client, error) {
    cmd := exec.Command(cfg.Command, cfg.Args...)
    // Set environment
    for k, v := range cfg.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    
    stdin, err := cmd.StdinPipe()
    if err != nil { return nil, err }
    
    stdout, err := cmd.StdoutPipe()
    if err != nil { return nil, err }
    
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("start MCP server %q: %w", name, err)
    }
    
    scanner := bufio.NewScanner(stdout)
    scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB lines
    
    client := &Client{
        cmd:    cmd,
        stdin:  stdin,
        stdout: scanner,
        name:   name,
    }
    
    // Initialize the MCP connection
    if err := client.initialize(); err != nil {
        client.Close()
        return nil, err
    }
    
    return client, nil
}

// Call sends a JSON-RPC request and waits for the response.
func (c *Client) Call(method string, params interface{}) (json.RawMessage, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.nextID++
    req := Request{
        JSONRPC: "2.0",
        ID:      c.nextID,
        Method:  method,
        Params:  params,
    }
    
    data, err := json.Marshal(req)
    if err != nil { return nil, err }
    
    // Write request + newline
    if _, err := c.stdin.Write(append(data, '\n')); err != nil {
        return nil, fmt.Errorf("write to MCP server: %w", err)
    }
    
    // Read response
    if !c.stdout.Scan() {
        return nil, fmt.Errorf("MCP server closed connection")
    }
    
    var resp Response
    if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
        return nil, fmt.Errorf("parse MCP response: %w", err)
    }
    
    if resp.Error != nil {
        return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
    }
    
    return resp.Result, nil
}

// ListTools discovers available tools from the MCP server.
func (c *Client) ListTools() ([]MCPTool, error) {
    result, err := c.Call("tools/list", nil)
    if err != nil { return nil, err }
    
    var resp struct {
        Tools []MCPTool `json:"tools"`
    }
    if err := json.Unmarshal(result, &resp); err != nil {
        return nil, err
    }
    return resp.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(name string, args json.RawMessage) (*MCPToolResult, error) {
    params := map[string]interface{}{
        "name":      name,
        "arguments": json.RawMessage(args),
    }
    result, err := c.Call("tools/call", params)
    if err != nil { return nil, err }
    
    var toolResult MCPToolResult
    if err := json.Unmarshal(result, &toolResult); err != nil {
        return nil, err
    }
    return &toolResult, nil
}

// Close shuts down the MCP server.
func (c *Client) Close() error {
    c.stdin.Close()
    return c.cmd.Process.Kill()
}

// initialize sends the MCP initialization handshake.
func (c *Client) initialize() error {
    _, err := c.Call("initialize", map[string]interface{}{
        "protocolVersion": "2024-11-05",
        "capabilities":    map[string]interface{}{},
        "clientInfo": map[string]string{
            "name":    "fev",
            "version": "0.1.0",
        },
    })
    return err
}
```

### Tests
- Use a mock MCP server (a simple Go program that reads stdin, writes stdout)
- Or test the JSON serialization/deserialization directly
- Test: Request marshals correctly, Response unmarshals correctly
- Test: RPCError is detected

### Commit
```bash
git commit -m "feat: MCP JSON-RPC client over stdin/stdout"
```

---

## Task 5: MCP Discovery

**What you're building:** Finding and parsing `.mcp.json` configuration files.

**Files:** `internal/mcp/discovery.go`, `internal/mcp/discovery_test.go`

### What to implement

```go
package mcp

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// DiscoverServers finds .mcp.json files and returns server configs.
// Searches in order:
// 1. Project root (current directory)
// 2. ~/.fev/mcp.json (global config)
func DiscoverServers(projectRoot, homeDir string) (map[string]ServerConfig, error) {
    allServers := make(map[string]ServerConfig)
    
    paths := []string{
        filepath.Join(projectRoot, ".mcp.json"),
        filepath.Join(homeDir, ".fev", "mcp.json"),
    }
    
    for _, path := range paths {
        servers, err := loadMCPConfig(path)
        if err != nil {
            continue // file doesn't exist or is invalid — skip
        }
        for name, cfg := range servers {
            if _, exists := allServers[name]; !exists {
                allServers[name] = cfg // project-level takes precedence
            }
        }
    }
    
    return allServers, nil
}

func loadMCPConfig(path string) (map[string]ServerConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var cfg MCPConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    
    return cfg.Servers, nil
}
```

### Tests
- Create temp .mcp.json, discover it
- Project-level overrides global
- Missing files are silently skipped
- Invalid JSON returns empty (not error)

### Commit
```bash
git commit -m "feat: MCP server discovery from .mcp.json"
```

---

## Task 6: MCP Bridge and Integration

**What you're building:** The glue that connects MCP tools to Fev's tool registry, plus wiring everything into main.go.

**Files:** `internal/mcp/bridge.go`, `internal/mcp/bridge_test.go`, modify `cmd/fev/main.go`

### What to implement

```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/crussella0129/fev/internal/permission"
    "github.com/crussella0129/fev/internal/tools"
)

// Bridge connects MCP servers to Fev's tool registry.
type Bridge struct {
    clients map[string]*Client  // server name → client
}

// NewBridge starts all discovered MCP servers and connects to them.
// Servers that fail to start are logged and skipped (never block startup).
func NewBridge(servers map[string]ServerConfig) *Bridge {
    bridge := &Bridge{
        clients: make(map[string]*Client),
    }
    
    for name, cfg := range servers {
        client, err := NewClient(name, cfg)
        if err != nil {
            log.Printf("warning: MCP server %q failed to start: %v", name, err)
            continue  // skip — never block startup
        }
        bridge.clients[name] = client
    }
    
    return bridge
}

// RegisterTools discovers tools from all MCP servers and registers them.
// Tools are registered with the prefix "mcp_{server}_{tool}".
func (b *Bridge) RegisterTools(registry *tools.Registry) error {
    for serverName, client := range b.clients {
        mcpTools, err := client.ListTools()
        if err != nil {
            log.Printf("warning: failed to list tools from MCP server %q: %v", serverName, err)
            continue
        }
        
        for _, mt := range mcpTools {
            toolName := fmt.Sprintf("mcp_%s_%s", serverName, mt.Name)
            wrapper := &mcpToolWrapper{
                name:        toolName,
                description: mt.Description,
                schema:      mt.InputSchema,
                client:      client,
                mcpToolName: mt.Name,
            }
            registry.Register(wrapper)
        }
    }
    return nil
}

// Close shuts down all MCP servers.
func (b *Bridge) Close() {
    for _, client := range b.clients {
        client.Close()
    }
}

// mcpToolWrapper adapts an MCP tool to Fev's Tool interface.
type mcpToolWrapper struct {
    name        string
    description string
    schema      json.RawMessage
    client      *Client
    mcpToolName string
}

func (t *mcpToolWrapper) Name() string                    { return t.name }
func (t *mcpToolWrapper) Description() string             { return t.description }
func (t *mcpToolWrapper) Schema() json.RawMessage         { return t.schema }
func (t *mcpToolWrapper) PermissionTier() permission.PermissionTier { 
    return permission.ReadOnly // MCP tools are read-only by default
}

func (t *mcpToolWrapper) Execute(ctx context.Context, args json.RawMessage) (*tools.ToolResult, error) {
    result, err := t.client.CallTool(t.mcpToolName, args)
    if err != nil {
        return nil, err
    }
    
    // Concatenate all text content
    var output string
    for _, c := range result.Content {
        if c.Type == "text" {
            output += c.Text + "\n"
        }
    }
    
    if result.IsError {
        return nil, fmt.Errorf("MCP tool error: %s", output)
    }
    
    return &tools.ToolResult{Output: output}, nil
}
```

### Integration into main.go

```go
// In runInteractive(), after creating the tool registry:

// MCP bridge
homeDir, _ := os.UserHomeDir()
mcpServers, _ := mcp.DiscoverServers(ws.Root(), homeDir)
var mcpBridge *mcp.Bridge
if len(mcpServers) > 0 {
    mcpBridge = mcp.NewBridge(mcpServers)
    mcpBridge.RegisterTools(reg)
    defer mcpBridge.Close()
}
```

### Tests
- mcpToolWrapper implements Tool interface
- Bridge skips failed servers (doesn't panic)
- RegisterTools adds tools with correct prefix

### Commit
```bash
git commit -m "feat: MCP bridge — discover servers, register tools"
git tag v0.4.0
```

---

## How to Verify the Whole Plan

After all 6 tasks:

```bash
cd C:/Users/charl/Fev
go test ./... -timeout 120s    # All tests pass
go build ./cmd/fev             # Builds clean
./bin/fev version              # fev v0.4.0
```

Then manually test:
1. Start Fev, try `/plan refactor this function` — should see a plan, approve/reject
2. Create a `.mcp.json` in a project directory, start Fev — should see MCP tools discovered
3. Test subagents by asking Fev to research multiple files in parallel (requires an LLM that can request parallel execution)

---

## What You'll Have After All 4 Plans

A complete `fev` binary that:
- Chats with any local LLM via OpenAI-compatible API
- Executes 10+ structured tools with safety checks
- Remembers facts across sessions (SQLite)
- Manages context intelligently (auto-compaction with circuit breaker)
- Displays polished terminal UI (colors, diffs, markdown, spinners)
- Spawns parallel research agents
- Shows plans for approval before executing
- Connects to MCP servers for external tool access
- Reads CLAUDE.md for ecosystem compatibility
- Ships as a single static binary on 5 platforms

**That's Fev.** A local-first Claude Code alternative that you own.
