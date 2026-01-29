package builtin

import (
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
)

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

func getStringArg(key string, args map[string]any) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}

	stringValue, ok := value.(string)
	return stringValue, ok
}

func RegisterAll(registry *tools.Registry, baseDir string) {
	RegisterCalculator(registry)
	RegisterReadFile(registry, baseDir)
	RegisterListFiles(registry, baseDir)
}
