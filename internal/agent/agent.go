// Package agent implements the flat agentic loop: generate → tool? → execute → loop.
package agent

import (
	"context"

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
