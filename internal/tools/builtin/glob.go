package builtin

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
)

const defaultMaxGlobResults = 1000

type Glob struct {
	BaseDir string
}

func (tool *Glob) Name() string { return "glob" }
func (tool *Glob) Description() string {
	return "Finds files matching a glob pattern. Supports ** for recursive matching."
}
func (tool *Glob) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match (e.g., '**/*.go', 'src/**/*.ts')",
			},
			"include_dirs": map[string]any{
				"type":        "boolean",
				"description": "Include directories in results (default: false)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 1000)",
			},
		},
		"required": []string{"pattern"},
	}
}
func (tool *Glob) RequiresApproval() bool { return false }

func (tool *Glob) Execute(args map[string]any, ctx context.Context) (string, error) {
	pattern, ok := getStringArg("pattern", args)
	if !ok || pattern == "" {
		return "", errors.New("missing pattern")
	}

	if !isSafeRelative(pattern) {
		return "", errors.New("absolute or parent paths are not allowed")
	}

	includeDirs, _ := getBoolArg("include_dirs", args)
	maxResults, hasMax := getIntArg("max_results", args)
	if !hasMax || maxResults <= 0 {
		maxResults = defaultMaxGlobResults
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)

	if strings.Contains(pattern, "**") {
		return tool.recursiveGlob(baseDir, pattern, includeDirs, maxResults)
	}

	return tool.simpleGlob(baseDir, pattern, includeDirs, maxResults)
}

func (tool *Glob) simpleGlob(baseDir, pattern string, includeDirs bool, maxResults int) (string, error) {
	fullPattern := filepath.Join(baseDir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return "", err
	}

	var results []string
	for _, match := range matches {
		if len(results) >= maxResults {
			break
		}

		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.IsDir() && !includeDirs {
			continue
		}

		relativePath, _ := filepath.Rel(baseDir, match)
		results = append(results, relativePath)
	}

	return strings.Join(results, "\n"), nil
}

func (tool *Glob) recursiveGlob(baseDir, pattern string, includeDirs bool, maxResults int) (string, error) {
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return "", errors.New("only one ** is supported in pattern")
	}

	prefix := strings.TrimSuffix(parts[0], string(os.PathSeparator))
	suffix := strings.TrimPrefix(parts[1], string(os.PathSeparator))

	searchDir := baseDir
	if prefix != "" {
		searchDir = filepath.Join(baseDir, prefix)
	}

	var results []string
	err := filepath.WalkDir(searchDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		if entry.IsDir() && !includeDirs {
			return nil
		}

		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}

		if suffix == "" {
			results = append(results, relativePath)
			return nil
		}

		matched, err := filepath.Match(suffix, filepath.Base(path))
		if err != nil {
			return nil
		}
		if matched {
			results = append(results, relativePath)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return strings.Join(results, "\n"), nil
}

func RegisterGlob(registry *tools.Registry, baseDir string) {
	registry.Add(&Glob{BaseDir: baseDir})
}
