package builtin

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/tools"
)

func resolveBaseDir(ctx context.Context, fallback string) string {
	if dir := tools.WorkingDir(ctx); dir != "" {
		return dir
	}
	return fallback
}

func isSafeRelative(path string) bool {
	if path == "" {
		return false
	}

	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return false
	}

	clean := filepath.Clean(path)
	return !strings.HasPrefix(clean, "..")
}

func validatePath(args map[string]any) (string, error) {
	path, ok := getStringArg("path", args)
	if !ok || path == "" {
		return "", errors.New("missing required argument: path")
	}

	if !isSafeRelative(path) {
		return "", errors.New("absolute or parent-traversal paths are not allowed")
	}

	return path, nil
}

func getStringArg(key string, args map[string]any) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}

	stringValue, ok := value.(string)
	return stringValue, ok
}

func getIntArg(key string, args map[string]any) (int, bool) {
	value, ok := args[key]
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func RegisterAll(registry *tools.Registry, baseDir string, toolsConfig config.ToolsConfig) {
	RegisterReadFile(registry, baseDir)
	RegisterListFiles(registry, baseDir)
	RegisterWriteFile(registry, baseDir, toolsConfig.File)
	RegisterEditFile(registry, baseDir, toolsConfig.File)
	RegisterWebFetch(registry, toolsConfig.Web)
}
