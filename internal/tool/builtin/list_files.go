package builtin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	toolpkg "github.com/erg0nix/kontekst/internal/tool"
)

// ListFiles is a tool that lists files matching a glob pattern.
type ListFiles struct {
	BaseDir string
}

func (tool *ListFiles) Name() string { return "list_files" }
func (tool *ListFiles) Description() string {
	return "Lists files matching a glob pattern. Returns relative paths, excludes directories."
}
func (tool *ListFiles) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string"},
		},
		"required": []string{"pattern"},
	}
}
func (tool *ListFiles) RequiresApproval() bool { return true }

func (tool *ListFiles) Execute(args map[string]any, ctx context.Context) (string, error) {
	pattern, ok := getStringArg("pattern", args)

	if !ok {
		return "", errors.New("missing pattern")
	}

	if !isSafeRelative(pattern) {
		return "", errors.New("absolute or parent paths are not allowed")
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPattern := filepath.Join(baseDir, pattern)
	matches, err := filepath.Glob(fullPattern)

	if err != nil {
		return "", err
	}

	var out []string

	for _, m := range matches {
		info, err := os.Stat(m)

		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		relativePath, _ := filepath.Rel(baseDir, m)
		out = append(out, relativePath)
	}

	return strings.Join(out, "\n"), nil
}

// RegisterListFiles adds the list_files tool to the registry.
func RegisterListFiles(registry *toolpkg.Registry, baseDir string) {
	registry.Add(&ListFiles{BaseDir: baseDir})
}
