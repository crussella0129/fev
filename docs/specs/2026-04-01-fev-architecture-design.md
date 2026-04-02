# Fev Architecture Design Spec

**Author:** Charles Russell / Thread & Signal LLC
**Date:** 2026-04-01
**Status:** Approved
**License:** MIT

---

## 1. Overview

Fev is a local-first agentic CLI harness written in 100% Go. It is a personal assistant for navigating the digital space — model-agnostic (any OpenAI-compatible API), single binary, zero runtime dependencies.

Fev is ecosystem-compatible: it reads CLAUDE.md files, uses the same tool names as Claude Code, and speaks MCP for shared tool access. It works alongside Claude Code rather than replacing it.

**Module:** `github.com/crussella0129/fev`
**Directory:** `C:/Users/charl/Fev`

**Heritage:** Informed by patterns from Animus Prion (Go), Claude Code (TypeScript), Claw-Code (Rust), and OpenClaude (TypeScript/Bun).

---

## 2. Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, goroutines for subagents, fast startup, easy cross-compilation |
| LLM Backend | OpenAI-compatible API only | Covers Ollama, llama.cpp server, vLLM, LM Studio. Users bring their own server. |
| Planning | Flat agent loop + `/plan` mode | Proven at scale (Claude Code). Plan mode adds user approval gate when needed. |
| Tools | 10 core tools | read_file, write_file, edit_file, bash, grep, glob, list_dir, git (status/diff/log/add/commit) |
| Memory | Full memory-as-hint | Session extraction, facts/hunches/corrections in SQLite, verification protocol, post-session review |
| Compaction | Full production pattern | Threshold-triggered, session memory summary, tool-pair invariant preservation, circuit breaker (3 strikes) |
| UI | Full Charm stack | bubbletea app shell, lipgloss styling, glamour markdown, bubbles components |
| Subagents | Goroutine-based | Read-only tools, background execution, timeout collection, single depth |
| Ecosystem | Convention + MCP | Reads CLAUDE.md, same tool names, MCP client for shared tool access |
| SQLite | modernc.org/sqlite | Pure Go, no CGo — cross-compilation stays trivial |

---

## 3. Project Structure

```
fev/
├── cmd/
│   └── fev/
│       └── main.go                    # Entrypoint (cobra CLI)
│
├── internal/
│   ├── agent/
│   │   ├── agent.go                   # Core flat agent loop
│   │   ├── planmode.go                # /plan command — propose before executing
│   │   ├── subagent.go                # Goroutine-spawned parallel agents
│   │   └── loop.go                    # Step execution: generate -> tool? -> observe -> loop
│   │
│   ├── context/
│   │   ├── manager.go                 # Token budget tracking + threshold triggers
│   │   ├── compact.go                 # Auto-compaction with circuit breaker
│   │   ├── session_memory.go          # Post-turn extraction (OpenClaude pattern)
│   │   ├── invariants.go              # Tool-pair preservation during compaction
│   │   └── blocks.go                  # Content block types
│   │
│   ├── memory/
│   │   ├── store.go                   # SQLite-backed persistent memory
│   │   ├── facts.go                   # Grounded, verified claims (source + timestamp)
│   │   ├── hunches.go                 # Unverified beliefs
│   │   ├── corrections.go             # "I was wrong about X" records
│   │   ├── session.go                 # Per-session structured log
│   │   ├── verify.go                  # Memory-as-hint verification protocol
│   │   └── review.go                  # Post-session summary on exit
│   │
│   ├── tools/
│   │   ├── registry.go                # Tool registration + JSON Schema validation
│   │   ├── executor.go                # Execution with timeout, safety, output-to-disk
│   │   ├── read_file.go
│   │   ├── write_file.go
│   │   ├── edit_file.go               # Surgical str_replace (Claude Code pattern)
│   │   ├── bash.go                    # Shell execution with restrictions
│   │   ├── grep.go                    # Dedicated ripgrep wrapper
│   │   ├── glob.go                    # File discovery
│   │   ├── list_dir.go
│   │   └── git.go                     # status, diff, log, add, commit
│   │
│   ├── llm/
│   │   ├── client.go                  # OpenAI-compatible HTTP client
│   │   ├── streaming.go               # SSE stream parsing
│   │   └── tokens.go                  # Token counting/estimation
│   │
│   ├── mcp/
│   │   ├── client.go                  # MCP client (connects to .mcp.json servers)
│   │   ├── discovery.go               # Find and load .mcp.json configs
│   │   └── bridge.go                  # Register MCP tools into Fev's tool registry
│   │
│   ├── config/
│   │   ├── config.go                  # FEV.md + CLAUDE.md + ~/.fev/config.yaml
│   │   └── detect.go                  # Hardware/model autodetection
│   │
│   ├── permission/
│   │   ├── checker.go                 # Deny-lists, injection detection, metachar blocking
│   │   └── policy.go                  # Permission tiers (ReadOnly, Write, Dangerous)
│   │
│   └── ui/
│       ├── app.go                     # Bubbletea application model
│       ├── terminal.go                # Lipgloss styled output
│       ├── spinner.go                 # Creative loading messages (bubbles)
│       ├── diff.go                    # Colorized unified diffs
│       ├── markdown.go                # Terminal markdown rendering (glamour)
│       └── verbs.go                   # Spinner verb collection
│
├── FEV.md                             # Per-project config template
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 4. Agent Loop

### 4.1 Default Mode (Flat Loop)

```
User Input
  -> Assemble context (system prompt, FEV.md/CLAUDE.md, memory, tool schemas, history)
  -> Auto-compact if over threshold
  -> Send to LLM
  -> Response contains tool calls?
     Yes -> validate schema -> check permissions -> execute -> feed result back -> loop
     No  -> render response to user -> done
  -> Max turns exceeded? -> warn and stop
```

### 4.2 Plan Mode (/plan)

```
User Input
  -> Same context assembly
  -> LLM generates a plan (text only, no tool execution)
  -> Display plan to user
  -> User approves?
     Yes -> execute plan (flat loop with approved tool set)
     No  -> revise or abandon
```

### 4.3 Key Behaviors

| Behavior | Source | Implementation |
|----------|--------|----------------|
| Tool call deduplication | Prion | Hash tool calls, skip identical consecutive calls |
| Repeat detection | Prion | 3x identical successful call -> break loop |
| Rate limiting | Prion | Progressive sleep between turns |
| Error classification | Prion | Retryable (exponential backoff) vs fatal (stop) |
| Tool description enforcement | Claude Code | Every bash call requires a `description` field |
| Large output to disk | Claude Code | Output > 8K tokens -> write to temp file, return preview + path |
| Streaming | Prion + Claude Code | SSE parsing, token-by-token output via bubbletea |
| Plan mode approval | Claude Code | /plan shows proposed actions, user confirms before execution |

---

## 5. Memory System

### 5.1 Layer 1: Session Memory (within a conversation)

Post-turn hook (OpenClaude pattern). After each LLM turn:
1. Check thresholds: context > 10K tokens AND grown by 5K since last extraction AND 3+ tool calls since last extraction
2. If all met -> ask the LLM (cheap call, ~500 tokens) to extract structured notes
3. Write to `~/.fev/sessions/{session_id}/session_memory.md`
4. This summary becomes the compaction replacement when auto-compact fires

**Session memory structure:**
```markdown
## Observations
- Grounded facts from file reads and tool outputs

## Decisions
- What was decided and why

## Open Questions
- Unresolved items

## Plan
- Current task state with checkboxes
```

### 5.2 Layer 2: Persistent Memory (across sessions, SQLite)

Three tables in `~/.fev/memory.db`:

| Table | Schema | Purpose |
|-------|--------|---------|
| `facts` | id, claim, source_file, source_line, created_at, verified_at | Grounded claims tied to source + timestamp |
| `hunches` | id, belief, confidence, created_at | Unverified beliefs |
| `corrections` | id, wrong_claim, right_claim, created_at | Explicit self-corrections |

### 5.3 Verification Protocol (Memory-as-Hint)

Before acting on any remembered fact:
1. Query `facts` table for a match
2. If found, check if `source_file` has changed since `verified_at` (stat mtime)
3. If changed or not found -> re-read the source file
4. If source contradicts memory -> update `facts` table AND insert a `corrections` record
5. Only then act on the information

### 5.4 Layer 3: Post-Session Review

Runs synchronously when user types `/exit` or `/quit`:
1. Scan session log for unresolved questions
2. Scan for corrections made during session
3. Promote confident hunches to facts (if verified during session)
4. Ask the LLM for a one-sentence session summary
5. Store in `~/.fev/sessions/{session_id}/review.md`
6. Next session loads most recent review as warm-start context

---

## 6. Context Window Management & Auto-Compaction

### 6.1 Token Budget Tracking

```
ContextManager:
    MaxTokens       int       // Detected from model API or config
    ReserveTokens   int       // Always keep free for generation (default: 2048)
    CompactAt       float64   // Trigger at this % full (default: 0.75)
    StaticContext   []Block   // System prompt, FEV.md, tool schemas — never compacted
    ActiveContext   []Block   // Recent conversation + tool results — compactable
    Archive         []Block   // Compacted summaries — read-only reference
    CircuitBreaker  int       // Consecutive failures (max 3)
```

### 6.2 Auto-Compaction Flow

After every LLM turn:
1. Estimate total tokens (static + active + archive)
2. If usage < CompactAt threshold -> do nothing
3. If CircuitBreaker >= 3 -> skip compaction, truncate oldest blocks, log what was lost
4. Otherwise:
   a. Wait for in-progress session memory extraction (15s timeout)
   b. Use session_memory.md as compaction summary (if available)
      - If no session memory yet, ask LLM to summarize oldest 50% of ActiveContext
   c. Find compaction boundary in ActiveContext
   d. Preserve tool-pair invariants (never split tool_use/tool_result)
   e. Replace compacted messages with summary block
   f. Move summary to Archive
   g. If compaction fails -> increment CircuitBreaker, fall through to truncation

### 6.3 Tool-Pair Invariant Preservation

Before discarding any messages during compaction:
1. Collect all `tool_result` message IDs in the keep-range
2. For each, verify the corresponding `tool_use` is also in the keep-range
3. If a `tool_use` would be discarded but its `tool_result` is kept -> expand keep-range backward
4. Never leave an orphaned tool_result or tool_use

### 6.4 Token Estimation

Simple heuristic (4 chars ~ 1 token) for speed. Optional tiktoken-go for accuracy when tokenizer is known. The estimate triggers compaction before the model's hard limit, not after.

---

## 7. Tool System

### 7.1 Tool Interface

```
Tool:
    Name        string
    Description string              // Fed to LLM for tool selection
    Schema      json.RawMessage     // JSON Schema for input validation
    Execute     func(input) -> (ToolResult, error)
    Permission  PermissionTier      // ReadOnly | Write | Dangerous
    Timeout     time.Duration

ToolResult:
    Output      string              // Text fed back to LLM
    Truncated   bool                // Was output cut short?
    FilePath    string              // If output too large, written here instead
    ExitCode    int                 // For bash tool
    Duration    time.Duration
    Diff        *StructuredDiff     // For edit/write tools — rendered by UI
```

### 7.2 Permission Tiers

| Tier | Tools | Behavior |
|------|-------|----------|
| ReadOnly | read_file, grep, glob, list_dir, git status/diff/log | Execute immediately |
| Write | write_file, edit_file, git add/commit | Show diff, execute (configurable confirmation) |
| Dangerous | bash | Show command + description, require confirmation |

### 7.3 Core Tools (10)

| Tool | Key Behaviors |
|------|--------------|
| read_file | Pagination (offset/limit). Dedup: skip if unchanged since last read (stat mtime). 10MB limit. Line numbers. |
| write_file | Absolute paths only. Structured diff in result. Creates parent dirs. |
| edit_file | old_string/new_string surgical replacement. Fails if old_string not unique. replace_all flag. |
| bash | Requires description field. List-based subprocess (no shell=True). Metachar rejection. Blocked commands. 120s default timeout (max 600s). Background support. |
| grep | Wraps ripgrep. Regex support. File type filter. Context lines. Head limit (default 250). Output modes: content, files_with_matches, count. |
| glob | Pattern matching. Sorted by mtime. Truncates at 100 files. |
| list_dir | Depth-limited. Ignores .git, node_modules, vendor. |
| git_status | Branch, modified files, untracked files. |
| git_diff | Staged and unstaged changes. |
| git_log | Recent commits with configurable count. |
| git_add | Stage specific files. |
| git_commit | Commit with message. |

### 7.4 Large Output Handling

When any tool output exceeds 8K tokens:
1. Write full output to `~/.fev/sessions/{session_id}/tool-results/{tool}_{timestamp}.txt`
2. Return first 2K tokens as preview + file path to the LLM
3. LLM can use read_file on the tool-results path if it needs more

### 7.5 MCP Bridge

Tools discovered via .mcp.json are registered with `mcp_{server}_{tool}` prefix. Same permission/validation/execution pipeline as native tools.

---

## 8. Subagent System

### 8.1 Lifecycle

```
Parent decides to parallelize
  -> Spawn N goroutines, each with:
     - Own agent loop (flat loop, independent conversation)
     - Copy of parent's file cache (snapshot, read-only)
     - Restricted tool registry (read-only tools only)
     - Own context window + token budget
     - Cancellable context with timeout
  -> Parent waits on result channel
     - Collects results as they arrive
     - 60s default timeout per subagent
     - If timeout -> cancel context, use partial results
  -> Parent synthesizes results into its context
  -> Parent continues its loop
```

### 8.2 Restrictions

| Capability | Parent | Subagent |
|------------|--------|----------|
| read_file, grep, glob, list_dir | Yes | Yes |
| git status, diff, log | Yes | Yes |
| write_file, edit_file | Yes | No |
| bash | Yes | No |
| git add, commit | Yes | No |
| Spawn sub-subagents | Yes | No (single depth) |
| MCP tools | Yes | Read-only MCP tools only |

### 8.3 Background Execution

Subagents can run in background:
- Parent continues its loop without waiting
- Results written to `~/.fev/sessions/{session_id}/subagent_{id}.json`
- Parent checks results later via internal check_subagent action
- UI notification when background subagent completes

---

## 9. UI System

### 9.1 Bubbletea Application

| View | Purpose |
|------|---------|
| Input | Multi-line text input with history recall |
| Response | Streaming markdown rendering, token-by-token |
| Diff | Colorized unified diff for edit_file/write_file |
| Spinner | Creative loading messages during LLM gen and tool execution |
| Plan | Plan mode display — numbered steps, approve/reject |
| Status | Bottom bar — model name, token usage, session duration, subagent status |
| Confirm | Permission prompts for Write/Dangerous tools |

### 9.2 Styling

- Lipgloss for colors, borders, headers
- Glamour for terminal markdown rendering
- Distinct visual treatment for: user input, assistant response, tool calls, tool results, errors, system messages
- Color theme configurable (dark/light/custom)
- Diff rendering: red/green with +/- prefixes, file headers, line numbers

### 9.3 Spinner Verbs

Creative loading messages, randomly selected per tool category:
- LLM generation: "Pondering...", "Reasoning through this...", "Thinking it over..."
- File reads: "Reading the source...", "Scanning the codebase..."
- Grep/glob: "Hunting for matches...", "Sifting through the project..."
- Bash: "Running the command...", "Executing..."
- Compaction: "Tidying up context...", "Making room to think..."
- Subagents: "Dispatching helpers...", "Gathering intel..."

---

## 10. Config & Ecosystem

### 10.1 Config Hierarchy (highest priority wins)

1. CLI flags (--model, --temperature, etc.)
2. FEV.md (project root)
3. CLAUDE.md (project root)
4. ~/.fev/config.yaml
5. Built-in defaults

### 10.2 Ecosystem Compatibility

- Reads CLAUDE.md natively (FEV.md takes precedence on conflicts)
- Tool names match Claude Code's conceptual model (read_file, edit_file, bash, grep, glob). LLMs familiar with Claude Code tool semantics work naturally with Fev.
- MCP client connects to same .mcp.json servers Claude Code uses. If an MCP server fails to start, Fev logs a warning and continues without those tools — never blocks startup.

### 10.3 CLI Commands

```
fev                    # Start interactive REPL
fev "do something"     # Single task, then interactive
fev init               # Create FEV.md template
fev config             # Show/edit config
fev version            # Print version
```

### 10.4 REPL Slash Commands

```
/plan       # Enter plan mode for next input
/exit       # Post-session review, then exit
/quit       # Same as /exit
/reset      # Clear conversation history
/memory     # Show memory stats
/compact    # Force manual compaction
/help       # Available commands
/model      # Switch model mid-session
```

---

## 11. Dependencies

| Purpose | Library | Notes |
|---------|---------|-------|
| CLI | github.com/spf13/cobra | Industry standard |
| Terminal app | github.com/charmbracelet/bubbletea | Interactive REPL |
| Styled output | github.com/charmbracelet/lipgloss | Colors, borders |
| Components | github.com/charmbracelet/bubbles | Spinners, input, viewport |
| Markdown | github.com/charmbracelet/glamour | Terminal markdown |
| SQLite | modernc.org/sqlite | Pure Go, no CGo |
| YAML | gopkg.in/yaml.v3 | Config parsing |
| Tokens | github.com/pkoukk/tiktoken-go | Optional accurate counting |
| Diffs | github.com/sergi/go-diff | Unified diffs |
| JSON Schema | github.com/santhosh-tekuri/jsonschema/v6 | Tool input validation |
| File watching | github.com/fsnotify/fsnotify | Cache invalidation |

No CGo. Pure Go. Cross-compiles to Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64).

---

## 12. Build & CI

```makefile
build:     go build -o bin/fev ./cmd/fev
test:      go test ./... -race -timeout 120s
lint:      golangci-lint run
release:   goreleaser release --snapshot --clean
```

GitHub Actions CI: test on Ubuntu + Windows + macOS. Release via goreleaser on tag push.

---

## 13. System Prompt

Fev's system prompt is injected as the first message in every conversation. It tells the LLM:
- Its identity (Fev, a local-first coding assistant)
- Available tools and their schemas
- The project context from FEV.md / CLAUDE.md
- Memory facts relevant to the current project
- Instructions for structured tool calling (JSON format)
- Permission model (which tools require confirmation)
- Constraints: stay within workspace boundary, respect Do Not rules from project config

The system prompt is part of StaticContext and is never compacted.
