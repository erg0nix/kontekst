package builtin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
)

const (
	defaultMaxGrepMatches   = 100
	defaultContextLines     = 0
	maxOutputBytesPerSearch = 1024 * 1024
)

type Grep struct {
	BaseDir string
}

func (tool *Grep) Name() string { return "grep" }
func (tool *Grep) Description() string {
	return "Searches file contents using regex patterns. Returns matching lines with file paths and line numbers."
}
func (tool *Grep) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory or file path to search in (default: current directory)",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "File pattern filter (e.g., '*.go', '*.ts')",
			},
			"context_lines": map[string]any{
				"type":        "integer",
				"description": "Number of context lines before and after match (default: 0)",
			},
			"max_matches": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matches to return (default: 100)",
			},
		},
		"required": []string{"pattern"},
	}
}
func (tool *Grep) RequiresApproval() bool { return false }

func (tool *Grep) Execute(args map[string]any, ctx context.Context) (string, error) {
	pattern, ok := getStringArg("pattern", args)
	if !ok || pattern == "" {
		return "", errors.New("missing pattern")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	searchPath, _ := getStringArg("path", args)
	if searchPath == "" {
		searchPath = "."
	}

	if !isSafeRelative(searchPath) {
		return "", errors.New("absolute or parent paths are not allowed")
	}

	globPattern, _ := getStringArg("glob", args)
	contextLines, hasContext := getIntArg("context_lines", args)
	if !hasContext || contextLines < 0 {
		contextLines = defaultContextLines
	}

	maxMatches, hasMax := getIntArg("max_matches", args)
	if !hasMax || maxMatches <= 0 {
		maxMatches = defaultMaxGrepMatches
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, searchPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", err
	}

	var results []string
	totalMatches := 0
	totalBytes := 0

	if info.IsDir() {
		err = filepath.WalkDir(fullPath, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if entry.IsDir() {
				return nil
			}

			if globPattern != "" {
				matched, _ := filepath.Match(globPattern, entry.Name())
				if !matched {
					return nil
				}
			}

			matches, matchCount, byteCount := searchFile(baseDir, path, re, contextLines, maxMatches-totalMatches)
			results = append(results, matches...)
			totalMatches += matchCount
			totalBytes += byteCount

			if totalMatches >= maxMatches || totalBytes >= maxOutputBytesPerSearch {
				return filepath.SkipAll
			}

			return nil
		})
	} else {
		matches, _, _ := searchFile(baseDir, fullPath, re, contextLines, maxMatches)
		results = append(results, matches...)
	}

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found", nil
	}

	return strings.Join(results, "\n"), nil
}

func searchFile(baseDir, path string, re *regexp.Regexp, contextLines int, maxMatches int) ([]string, int, int) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0
	}
	defer file.Close()

	relativePath, _ := filepath.Rel(baseDir, path)

	var results []string
	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	matchCount := 0
	byteCount := 0

	for lineNum, line := range lines {
		if matchCount >= maxMatches {
			break
		}

		if !re.MatchString(line) {
			continue
		}

		matchCount++

		if contextLines == 0 {
			result := fmt.Sprintf("%s:%d:%s", relativePath, lineNum+1, line)
			results = append(results, result)
			byteCount += len(result)
		} else {
			startLine := lineNum - contextLines
			if startLine < 0 {
				startLine = 0
			}
			endLine := lineNum + contextLines
			if endLine >= len(lines) {
				endLine = len(lines) - 1
			}

			var contextBlock strings.Builder
			for i := startLine; i <= endLine; i++ {
				prefix := " "
				if i == lineNum {
					prefix = ">"
				}
				contextBlock.WriteString(fmt.Sprintf("%s%s:%d:%s\n", prefix, relativePath, i+1, lines[i]))
			}
			result := strings.TrimSuffix(contextBlock.String(), "\n")
			results = append(results, result)
			byteCount += len(result)
		}
	}

	return results, matchCount, byteCount
}

func RegisterGrep(registry *tools.Registry, baseDir string) {
	registry.Add(&Grep{BaseDir: baseDir})
}
