# Plan 2: Memory & Compaction

> **Prerequisite:** Plan 1 (Foundation) must be complete — v0.1.0 with 85+ tests passing.
> **Goal:** Give Fev persistent memory across sessions and intelligent context window management so it doesn't lose track of long conversations.
> **Estimated effort:** 8 tasks, ~60 TDD steps

---

## What This Plan Builds

Right now, Fev forgets everything when you close it. This plan adds:

1. **Session memory** — Fev takes notes during a conversation (observations, decisions, open questions) and uses those notes as a summary when the context window fills up
2. **Persistent memory** — SQLite database storing facts, hunches, and corrections across sessions
3. **Auto-compaction** — When the conversation gets too long for the model's context window, Fev intelligently compresses older messages while preserving critical information
4. **Post-session review** — When you type `/exit`, Fev summarizes what it learned

---

## Go Concepts You'll Need

If you're new to Go, here's what matters for this plan:

### SQLite in Go (modernc.org/sqlite)
```go
import "database/sql"
import _ "modernc.org/sqlite" // Register the driver — the underscore means "import for side effects only"

db, err := sql.Open("sqlite", "path/to/memory.db")
// Use db.Exec() for writes, db.Query()/db.QueryRow() for reads
// Always defer db.Close()
// Always check err != nil after every database call
```

### Goroutines (lightweight threads)
```go
go func() {
    // This runs concurrently — doesn't block the main thread
    result := doExpensiveWork()
    channel <- result  // Send result back via a channel
}()
```

### Channels (typed pipes between goroutines)
```go
ch := make(chan string, 1)  // Buffered channel — holds 1 item without blocking
go func() { ch <- "done" }()
result := <-ch  // Blocks until something is sent
```

### Context with timeout
```go
ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
defer cancel()
select {
case result := <-ch:
    // Got result in time
case <-ctx.Done():
    // Timed out
}
```

### The `sync.Mutex` pattern
```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}
func (c *SafeCounter) Increment() {
    c.mu.Lock()         // Only one goroutine at a time
    defer c.mu.Unlock() // Always unlock, even if panic
    c.count++
}
```

---

## New Dependencies

Add to go.mod:
```bash
cd C:/Users/charl/Fev
go get modernc.org/sqlite@latest
```

This is a **pure Go** SQLite implementation — no C compiler needed. It's slower than the C version (`mattn/go-sqlite3`) but cross-compiles trivially.

---

## File Structure

```
internal/
  memory/
    store.go              # SQLite database setup, migrations, connection
    store_test.go
    facts.go              # Facts table — grounded claims with source + timestamp
    facts_test.go
    hunches.go            # Hunches table — unverified beliefs
    hunches_test.go
    corrections.go        # Corrections table — "I was wrong about X"
    corrections_test.go
    verify.go             # Memory-as-hint verification protocol
    verify_test.go
    session.go            # Session log (structured markdown file)
    session_test.go
    review.go             # Post-session review on /exit
    review_test.go
  context/                # RENAME from ctxwin/ to context/ for clarity
    manager.go            # Enhanced — add compaction threshold tracking
    compact.go            # Auto-compaction with circuit breaker
    compact_test.go
    session_memory.go     # Post-turn extraction (runs in background goroutine)
    session_memory_test.go
    invariants.go         # Tool-pair preservation during compaction
    invariants_test.go
```

---

## Task 1: SQLite Memory Store Setup

**What you're building:** The database layer that stores Fev's persistent memory.

**Files:** `internal/memory/store.go`, `internal/memory/store_test.go`

### Why this matters
Every time Fev reads a file or learns something about your project, it can store that as a "fact" in SQLite. Next session, it can look up those facts instead of re-reading everything. But — and this is the critical design decision — **Fev never trusts its own memory blindly**. It always verifies facts against the actual filesystem before acting on them. This is called "memory-as-hint."

### What to implement

**store.go:**
```go
package memory

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    _ "modernc.org/sqlite"
)

// Store is the persistent memory database.
type Store struct {
    db *sql.DB
}

// NewStore opens (or creates) the SQLite database at the given path.
// It runs migrations to create tables if they don't exist.
func NewStore(dbPath string) (*Store, error) {
    // Create parent directories
    if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
        return nil, fmt.Errorf("create db directory: %w", err)
    }
    
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }
    
    // Enable WAL mode for better concurrent read/write performance
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        db.Close()
        return nil, fmt.Errorf("set WAL mode: %w", err)
    }
    
    // Run migrations
    if err := migrate(db); err != nil {
        db.Close()
        return nil, fmt.Errorf("migrate: %w", err)
    }
    
    return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
    return s.db.Close()
}

// migrate creates tables if they don't exist.
func migrate(db *sql.DB) error {
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS facts (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            claim TEXT NOT NULL,
            source_file TEXT,
            source_line INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            verified_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS hunches (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            belief TEXT NOT NULL,
            confidence REAL DEFAULT 0.5,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS corrections (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            wrong_claim TEXT NOT NULL,
            right_claim TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        `CREATE INDEX IF NOT EXISTS idx_facts_source ON facts(source_file)`,
        `CREATE INDEX IF NOT EXISTS idx_facts_claim ON facts(claim)`,
    }
    
    for _, m := range migrations {
        if _, err := db.Exec(m); err != nil {
            return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
        }
    }
    return nil
}
```

### Tests to write
- `TestNewStore` — create in temp dir, verify no error, close
- `TestNewStore_CreatesDirectories` — use a nested path that doesn't exist yet
- `TestNewStore_MigratesOnOpen` — open, close, reopen — tables should still exist
- `TestClose` — open, close, verify no error

### How to run
```bash
cd C:/Users/charl/Fev
go test ./internal/memory/ -v -run TestNewStore
```

### Commit
```bash
git add internal/memory/store.go internal/memory/store_test.go
git commit -m "feat: SQLite memory store with WAL mode and migrations"
```

---

## Task 2: Facts Table

**What you're building:** The ability to store and query grounded facts — things Fev has verified by reading actual files.

**Files:** `internal/memory/facts.go`, `internal/memory/facts_test.go`

### What a "fact" is

A fact is a claim tied to evidence:
- **Claim:** "main.go imports net/http"  
- **Source:** `main.go`, line 5
- **Timestamp:** When Fev last verified this

Facts become stale when the source file changes. The verification protocol (Task 5) handles that.

### What to implement

```go
// Fact is a grounded, verified claim about the codebase.
type Fact struct {
    ID         int64
    Claim      string
    SourceFile string // file path that proves this claim
    SourceLine int    // line number (0 if unknown)
    CreatedAt  time.Time
    VerifiedAt time.Time
}

// InsertFact stores a new fact.
func (s *Store) InsertFact(claim, sourceFile string, sourceLine int) (int64, error)

// FindFacts searches for facts matching a query (LIKE %query%).
func (s *Store) FindFacts(query string, limit int) ([]Fact, error)

// FindFactsBySource returns all facts from a specific file.
func (s *Store) FindFactsBySource(sourceFile string) ([]Fact, error)

// UpdateFactVerification marks a fact as re-verified now.
func (s *Store) UpdateFactVerification(id int64) error

// DeleteFact removes a fact (used when verification shows it's wrong).
func (s *Store) DeleteFact(id int64) error
```

### Tests
- Insert a fact, find it by query
- Insert multiple facts, find by source file
- Update verification timestamp
- Delete a fact, verify it's gone
- FindFacts with no matches returns empty slice (not nil, not error)

### Commit
```bash
git commit -m "feat: facts table — grounded claims with source tracking"
```

---

## Task 3: Hunches and Corrections Tables

**Files:** `internal/memory/hunches.go`, `internal/memory/hunches_test.go`, `internal/memory/corrections.go`, `internal/memory/corrections_test.go`

### What hunches are

A hunch is an unverified belief: "this project probably uses PostgreSQL." Hunches have a confidence score (0.0 to 1.0). When a hunch is verified, it gets promoted to a fact. When it's disproven, it becomes a correction.

### What corrections are

A correction records: "I believed X, but actually Y." This trains the verification instinct — next time Fev sees a similar pattern, it's more likely to check.

### What to implement

```go
// Hunch is an unverified belief.
type Hunch struct {
    ID         int64
    Belief     string
    Confidence float64 // 0.0 to 1.0
    CreatedAt  time.Time
}

// InsertHunch stores a new hunch.
func (s *Store) InsertHunch(belief string, confidence float64) (int64, error)

// FindHunches searches for hunches matching a query.
func (s *Store) FindHunches(query string, limit int) ([]Hunch, error)

// DeleteHunch removes a hunch (when promoted to fact or disproven).
func (s *Store) DeleteHunch(id int64) error

// Correction records an explicit self-correction.
type Correction struct {
    ID         int64
    WrongClaim string
    RightClaim string
    CreatedAt  time.Time
}

// InsertCorrection records that a belief was wrong.
func (s *Store) InsertCorrection(wrongClaim, rightClaim string) (int64, error)

// RecentCorrections returns the N most recent corrections.
func (s *Store) RecentCorrections(limit int) ([]Correction, error)
```

### Tests
- Insert hunch with confidence, find it
- Delete hunch
- Insert correction, retrieve recent
- Confidence clamped to [0.0, 1.0]

### Commit
```bash
git commit -m "feat: hunches and corrections tables"
```

---

## Task 4: Session Log

**What you're building:** A structured markdown file that tracks what happened in the current session.

**Files:** `internal/memory/session.go`, `internal/memory/session_test.go`

### How it works

Each session gets a unique ID and a directory at `~/.fev/sessions/{session_id}/`. Inside that directory:
- `session_memory.md` — structured notes (observations, decisions, questions, plan)
- `review.md` — post-session summary (written on /exit)
- `tool-results/` — large tool outputs that were written to disk

### What to implement

```go
// Session tracks the current session state.
type Session struct {
    ID        string
    Dir       string    // ~/.fev/sessions/{id}/
    StartedAt time.Time
    
    Observations []string // things the agent has seen
    Decisions    []string // what was decided and why
    Questions    []string // unresolved items
    Plan         []string // current task state
}

// NewSession creates a new session with a unique ID.
func NewSession(baseDir string) (*Session, error)
// - ID = timestamp + random suffix (e.g., "20260401-143022-a7b3")
// - Creates the session directory
// - Creates tool-results/ subdirectory

// AddObservation records something the agent observed.
func (s *Session) AddObservation(obs string)

// AddDecision records a decision.
func (s *Session) AddDecision(dec string)

// AddQuestion records an unresolved question.
func (s *Session) AddQuestion(q string)

// SetPlan sets the current plan.
func (s *Session) SetPlan(items []string)

// Save writes the session memory to session_memory.md.
func (s *Session) Save() error
// Writes markdown in this format:
// ## Observations
// - item 1
// - item 2
// 
// ## Decisions
// - item 1
// ...

// Load reads session_memory.md back into the struct.
func LoadSession(dir string) (*Session, error)

// ToolResultPath returns a path for storing large tool output.
func (s *Session) ToolResultPath(toolName string) string
// Returns: s.Dir + "/tool-results/" + toolName + "_" + timestamp + ".txt"
```

### Tests
- NewSession creates directory
- Add observations, save, load — round-trip
- ToolResultPath returns valid path inside session dir

### Commit
```bash
git commit -m "feat: session log — structured markdown per session"
```

---

## Task 5: Memory-as-Hint Verification Protocol

**What you're building:** The protocol that ensures Fev never acts on stale memory without checking.

**Files:** `internal/memory/verify.go`, `internal/memory/verify_test.go`

### The protocol

Before acting on any remembered fact:
1. Query the `facts` table for a match
2. If found, `stat()` the source file — has it changed since `verified_at`?
3. If the file's modification time is after `verified_at` → the fact is stale
4. For stale facts: re-read the source file, check if the claim is still true
5. If still true → update `verified_at`
6. If false → delete the fact, insert a correction, return "stale"

### What to implement

```go
// VerificationResult tells the caller what happened.
type VerificationResult struct {
    Status    VerifyStatus
    Fact      *Fact    // the fact (if found and still valid)
    Stale     bool     // true if the fact was outdated
    Corrected bool     // true if a correction was recorded
}

type VerifyStatus int
const (
    VerifyNotFound VerifyStatus = iota // no matching fact in memory
    VerifyValid                         // fact exists and is current
    VerifyStale                         // fact exists but source file changed
)

// Verifier checks facts against the filesystem.
type Verifier struct {
    store *Store
}

func NewVerifier(store *Store) *Verifier

// Verify checks if a claim about a file is still true.
func (v *Verifier) Verify(claim, sourceFile string) (*VerificationResult, error)
// 1. FindFacts matching the claim
// 2. If no match → return VerifyNotFound
// 3. os.Stat the source file → get ModTime
// 4. If ModTime > fact.VerifiedAt → return VerifyStale
// 5. Else → update verified_at, return VerifyValid
```

### Tests
- Fact about existing unchanged file → VerifyValid
- Fact about modified file → VerifyStale
- No matching fact → VerifyNotFound
- Use `os.Chtimes` to manipulate file modification times in tests

### Commit
```bash
git commit -m "feat: memory-as-hint verification protocol"
```

---

## Task 6: Tool-Pair Invariant Preservation

**What you're building:** Logic that ensures compaction never breaks the conversation by splitting a tool call from its result.

**Files:** `internal/context/invariants.go`, `internal/context/invariants_test.go`

### The problem

The LLM sends tool calls, and Fev sends back results. These come in pairs:
```
[assistant: tool_calls=[{id:"call_1", name:"read_file", args:{...}}]]
[tool: tool_call_id="call_1", content="file contents..."]
```

If compaction removes the assistant message but keeps the tool result, the LLM gets confused — it sees a result for a call it never made. Similarly, if the tool call is kept but the result is removed, the LLM waits forever for a response.

### What to implement

```go
// FindCompactionBoundary finds the safe index to split messages for compaction.
// Messages before the index are compacted (summarized), messages at and after are kept.
// Guarantees: no orphaned tool_use or tool_result messages.
func FindCompactionBoundary(messages []core.Message, targetKeepCount int) int
// Walk backward from the end of messages.
// Count targetKeepCount messages.
// Then adjust: if the boundary would split a tool-use/tool-result pair,
// move the boundary backward to include both.

// ValidateToolPairs checks that all tool results have matching tool calls.
func ValidateToolPairs(messages []core.Message) error
// For each message with Role=tool and ToolCallID set,
// verify there's a preceding assistant message with a matching ToolCall.ID.
```

### How to identify pairs
- An **assistant message** with `ToolCalls` field populated = tool use
- A **tool message** with `ToolCallID` field set = tool result
- They're paired by the `ID`/`ToolCallID` match

### Tests
- Messages with no tool calls → boundary is just targetKeepCount from end
- Messages with tool pair at boundary → boundary adjusts to include both
- Multiple tool calls in one assistant message → all results must be kept together
- ValidateToolPairs detects orphaned result

### Commit
```bash
git commit -m "feat: tool-pair invariant preservation for compaction"
```

---

## Task 7: Auto-Compaction with Circuit Breaker

**What you're building:** The automatic context compression system that fires when the conversation gets too long.

**Files:** `internal/context/compact.go`, `internal/context/compact_test.go`, `internal/context/session_memory.go`, `internal/context/session_memory_test.go`

### How compaction works

1. After every LLM turn, check: is usage > 75% of max tokens?
2. If yes, and circuit breaker hasn't tripped (< 3 consecutive failures):
   a. Wait for any in-progress session memory extraction (max 15s)
   b. If session_memory.md exists, use it as the compaction summary
   c. Otherwise, ask the LLM to summarize the oldest 50% of messages
   d. Find the safe compaction boundary (Task 6)
   e. Replace old messages with the summary
   f. If this fails, increment circuit breaker
3. If circuit breaker >= 3: just truncate oldest messages and log what was lost

### Session memory extraction

This runs as a **background goroutine** after each LLM turn. It's separate from compaction:

```go
// Extractor runs session memory extraction in the background.
type Extractor struct {
    mu          sync.Mutex
    running     bool
    lastTokens  int     // token count at last extraction
    toolCalls   int     // tool calls since last extraction
    memoryPath  string  // path to session_memory.md
    
    // Thresholds (from OpenClaude)
    initThreshold   int  // 10000 tokens before first extraction
    growthThreshold int  // 5000 token growth between extractions
    callThreshold   int  // 3 tool calls between extractions
}

// MaybeExtract checks thresholds and runs extraction if all are met.
// This is called after every LLM turn.
func (e *Extractor) MaybeExtract(ctx context.Context, client agent.LLMClient, 
    messages []core.Message, currentTokens int)
// 1. Check: currentTokens > initThreshold?
// 2. Check: currentTokens - lastTokens > growthThreshold?
// 3. Check: toolCalls >= callThreshold?
// 4. If all met: spawn goroutine to extract, set running=true
// 5. Goroutine: ask LLM to summarize recent observations/decisions/questions
// 6. Write to memoryPath
// 7. Reset counters, set running=false

// WaitForCompletion blocks until extraction finishes or timeout.
func (e *Extractor) WaitForCompletion(timeout time.Duration) bool
```

### Circuit breaker

```go
type Compactor struct {
    circuitBreaker int  // consecutive failures
    maxFailures    int  // 3
}

// Compact attempts to compress the message history.
func (c *Compactor) Compact(ctx context.Context, messages []core.Message, 
    client agent.LLMClient, sessionMemoryPath string) ([]core.Message, error)
// 1. If circuitBreaker >= maxFailures → truncate (no LLM call)
// 2. Try to read session_memory.md as summary
// 3. If no session memory → ask LLM to summarize oldest 50%
// 4. Find safe boundary via FindCompactionBoundary
// 5. Replace old messages with summary message
// 6. On failure → increment circuitBreaker, fall back to truncation
// 7. On success → reset circuitBreaker
```

### Tests
- Compaction triggers at threshold
- Circuit breaker trips after 3 failures
- Session memory extraction respects all 3 thresholds
- Tool-pair preservation during compaction
- Truncation fallback works when circuit breaker is tripped

### Commit
```bash
git commit -m "feat: auto-compaction with circuit breaker, session memory extraction"
```

---

## Task 8: Post-Session Review and Integration

**What you're building:** The review that runs when you type `/exit`, plus wiring everything into the agent.

**Files:** `internal/memory/review.go`, `internal/memory/review_test.go`, modify `internal/agent/agent.go`, modify `cmd/fev/main.go`

### Post-session review

When the user exits:
1. Scan session log for unresolved questions
2. Scan for corrections made during session
3. Ask the LLM for a one-sentence session summary
4. Store in `~/.fev/sessions/{session_id}/review.md`
5. On next session start, load the most recent review as warm-start context

```go
// Review generates a post-session summary.
func Review(ctx context.Context, client agent.LLMClient, 
    session *Session, store *Store) (string, error)
// 1. Collect session.Questions (unresolved)
// 2. Query store.RecentCorrections(10)
// 3. Build a prompt: "Summarize this session in one sentence. 
//    Unresolved questions: [...] Corrections made: [...]"
// 4. Call client.Generate with this prompt
// 5. Write review to session.Dir + "/review.md"
// 6. Return the summary

// LoadLatestReview finds the most recent session's review.md.
func LoadLatestReview(baseDir string) (string, error)
// List session directories, sort by name (they're timestamp-prefixed),
// read the last one's review.md
```

### Integration into the agent

Modify `internal/agent/agent.go`:
- Add `memory *memory.Store` field
- Add `session *memory.Session` field  
- In `Run()`, after each tool execution, store file-read results as facts
- Pass memory context into the system prompt

Modify `cmd/fev/main.go`:
- Create Store on startup
- Create Session on startup
- Load latest review into system prompt
- On `/exit`, call Review() before shutting down

### Tests
- Review generates non-empty summary
- LoadLatestReview finds the right session
- Integration: agent stores facts from file reads

### Commit
```bash
git commit -m "feat: post-session review, memory integration into agent"
git tag v0.2.0
```

---

## How to Verify the Whole Plan

After all 8 tasks:

```bash
cd C:/Users/charl/Fev
go test ./... -timeout 120s    # All tests pass
go build ./cmd/fev             # Builds clean
./bin/fev version              # fev v0.2.0
```

Then manually test:
1. Start Fev, have a conversation, type `/exit` — review.md should appear in `~/.fev/sessions/`
2. Start Fev again — latest review should be in the system prompt
3. Have a long conversation — compaction should trigger automatically
4. Check `~/.fev/memory.db` — should contain facts from file reads
