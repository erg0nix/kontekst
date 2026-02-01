package builtin

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/erg0nix/kontekst/internal/tools"
)

type ReadFile struct {
	BaseDir string
}

func (tool *ReadFile) Name() string        { return "read_file" }
func (tool *ReadFile) Description() string { return "Reads a file" }
func (tool *ReadFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
		},
		"required": []string{"path"},
	}
}
func (tool *ReadFile) RequiresApproval() bool { return true }

func (tool *ReadFile) Execute(args map[string]any, ctx context.Context) (string, error) {
	path, ok := getStringArg("path", args)

	if !ok {
		return "", errors.New("missing path")
	}

	if !isSafeRelative(path) {
		return "", errors.New("absolute or parent paths are not allowed")
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)
	data, err := os.ReadFile(fullPath)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func RegisterReadFile(registry *tools.Registry, baseDir string) {
	registry.Add(&ReadFile{BaseDir: baseDir})
}
