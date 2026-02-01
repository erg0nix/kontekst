package tools

import (
	"context"
	"errors"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

type contextKey string

const workingDirKey contextKey = "workingDir"

func WithWorkingDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, workingDirKey, dir)
}

func WorkingDir(ctx context.Context) string {
	if dir, ok := ctx.Value(workingDirKey).(string); ok {
		return dir
	}
	return ""
}

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	RequiresApproval() bool
	Execute(args map[string]any, ctx context.Context) (string, error)
}

type ToolExecutor interface {
	Execute(name string, args map[string]any, ctx context.Context) (string, error)
	ToolDefinitions() []core.ToolDef
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (registry *Registry) Add(tool Tool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.tools[tool.Name()] = tool
}

func (registry *Registry) Execute(name string, args map[string]any, ctx context.Context) (string, error) {
	registry.mu.RLock()
	tool, ok := registry.tools[name]
	registry.mu.RUnlock()

	if !ok {
		return "", errors.New("tool not found")
	}

	return tool.Execute(args, ctx)
}

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
