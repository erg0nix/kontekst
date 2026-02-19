package tool

import (
	"context"
	"errors"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

type contextKey string

const workingDirKey contextKey = "workingDir"

// WithWorkingDir returns a new context carrying the given working directory.
func WithWorkingDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, workingDirKey, dir)
}

// WorkingDir extracts the working directory from the context, or returns empty string if unset.
func WorkingDir(ctx context.Context) string {
	if dir, ok := ctx.Value(workingDirKey).(string); ok {
		return dir
	}
	return ""
}

// Tool defines the interface that all agent tools must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	RequiresApproval() bool
	Execute(args map[string]any, ctx context.Context) (string, error)
}

// Previewer is an optional interface that tools can implement to show a preview before execution.
type Previewer interface {
	Preview(args map[string]any, ctx context.Context) (string, error)
}

// ToolExecutor is the interface used by the agent to execute tools and retrieve their definitions.
type ToolExecutor interface {
	Execute(name string, args map[string]any, ctx context.Context) (string, error)
	ToolDefinitions() []core.ToolDef
	Preview(name string, args map[string]any, ctx context.Context) (string, error)
}

// Registry is a thread-safe collection of named tool.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Add registers a tool in the registry, keyed by its name.
func (registry *Registry) Add(tool Tool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.tools[tool.Name()] = tool
}

// Execute runs the named tool with the given arguments and returns its output.
func (registry *Registry) Execute(name string, args map[string]any, ctx context.Context) (string, error) {
	registry.mu.RLock()
	tool, ok := registry.tools[name]
	registry.mu.RUnlock()

	if !ok {
		return "", errors.New("tool not found")
	}

	return tool.Execute(args, ctx)
}

// GetTool looks up a tool by name, returning it and whether it was found.
func (registry *Registry) GetTool(name string) (Tool, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	tool, ok := registry.tools[name]
	return tool, ok
}

// Preview returns a preview of what the named tool would do, or empty string if it doesn't support previewing.
func (registry *Registry) Preview(name string, args map[string]any, ctx context.Context) (string, error) {
	registry.mu.RLock()
	tool, ok := registry.tools[name]
	registry.mu.RUnlock()

	if !ok {
		return "", nil
	}

	if previewer, ok := tool.(Previewer); ok {
		return previewer.Preview(args, ctx)
	}

	return "", nil
}

// ToolDefinitions returns the LLM-facing definitions of all registered tool.
func (registry *Registry) ToolDefinitions() []core.ToolDef {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	definitions := make([]core.ToolDef, 0, len(registry.tools))

	for _, tool := range registry.tools {
		definitions = append(definitions, core.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}

	return definitions
}
