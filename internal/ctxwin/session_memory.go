package ctxwin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/crussella0129/fev/internal/core"
)

// Extractor runs session memory extraction in the background after each LLM turn.
// It asks the LLM to summarize observations/decisions/questions when all three
// thresholds are met: token growth, total token count, and tool call count.
type Extractor struct {
	mu         sync.Mutex
	running    bool
	lastTokens int
	toolCalls  int
	memoryPath string

	// Thresholds (from OpenClaude pattern)
	initThreshold   int // total tokens before first extraction (default 10000)
	growthThreshold int // token growth between extractions (default 5000)
	callThreshold   int // tool calls between extractions (default 3)

	done chan struct{} // closed when the current extraction finishes
}

// NewExtractor creates an Extractor that writes to memoryPath.
func NewExtractor(memoryPath string) *Extractor {
	return &Extractor{
		memoryPath:      memoryPath,
		initThreshold:   10000,
		growthThreshold: 5000,
		callThreshold:   3,
		done:            make(chan struct{}),
	}
}

// RecordToolCall increments the tool call counter.
// Call this whenever the agent executes a tool.
func (e *Extractor) RecordToolCall() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.toolCalls++
}

// MaybeExtract checks all three thresholds and spawns a background goroutine
// to extract session memory if all are satisfied.
//
// This is non-blocking: it returns immediately and extraction happens in the background.
func (e *Extractor) MaybeExtract(ctx context.Context, client Summarizer, messages []core.Message, currentTokens int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return // extraction already in progress
	}

	if currentTokens < e.initThreshold {
		return
	}
	if currentTokens-e.lastTokens < e.growthThreshold {
		return
	}
	if e.toolCalls < e.callThreshold {
		return
	}

	// All thresholds met — capture state for the goroutine.
	snapshot := make([]core.Message, len(messages))
	copy(snapshot, messages)
	e.running = true
	e.done = make(chan struct{})

	go func() {
		defer func() {
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			close(e.done)
		}()

		summary := extractSummary(ctx, client, snapshot)
		if err := os.WriteFile(e.memoryPath, []byte(summary), 0644); err != nil {
			return // best-effort; don't crash the agent
		}

		e.mu.Lock()
		e.lastTokens = currentTokens
		e.toolCalls = 0
		e.mu.Unlock()
	}()
}

// WaitForCompletion blocks until extraction finishes or the timeout elapses.
// Returns true if extraction completed, false if timeout was reached.
func (e *Extractor) WaitForCompletion(timeout time.Duration) bool {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return true
	}
	done := e.done
	e.mu.Unlock()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// extractSummary asks the LLM to produce session memory notes.
func extractSummary(ctx context.Context, client Summarizer, messages []core.Message) string {
	var b strings.Builder
	for _, m := range messages {
		if m.Role == core.RoleTool || m.Role == core.RoleSystem {
			continue
		}
		fmt.Fprintf(&b, "[%s]: %s\n", m.Role, m.Content)
	}

	prompt := "Extract structured session notes from this conversation. " +
		"Format your response as:\n" +
		"## Observations\n- ...\n\n## Decisions\n- ...\n\n## Questions\n- ...\n\n" +
		"Be concise. Only include items that would be useful context in a future session.\n\n" +
		b.String()

	result, err := client.Generate(ctx, []core.Message{core.NewUserMessage(prompt)}, nil)
	if err != nil {
		return "## Observations\n_(extraction failed)_\n"
	}
	return result.Content
}
