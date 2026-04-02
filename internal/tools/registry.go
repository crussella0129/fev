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
