package builtin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
)

const maxLinesDefault = 10000

type ReadFile struct {
	BaseDir string
}

func (tool *ReadFile) Name() string { return "read_file" }
func (tool *ReadFile) Description() string {
	return "Reads a file and returns its content with line numbers. Supports optional line range (1-indexed, inclusive)."
}
func (tool *ReadFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file to read",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "First line to read (1-indexed, default: 1)",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "Last line to read (inclusive, default: end of file)",
			},
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

	startLine, _ := getIntArg("start_line", args)
	if startLine <= 0 {
		startLine = 1
	}

	endLine, hasEndLine := getIntArg("end_line", args)

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		if lineNumber < startLine {
			continue
		}

		if hasEndLine && lineNumber > endLine {
			break
		}

		lines = append(lines, scanner.Text())

		if !hasEndLine && len(lines) >= maxLinesDefault {
			return "", fmt.Errorf("file has more than %d lines; specify start_line and end_line to read a range", maxLinesDefault)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(lines) == 0 {
		if hasEndLine {
			return "", fmt.Errorf("no lines in range %d-%d (file has %d lines)", startLine, endLine, lineNumber)
		}
		return "", fmt.Errorf("no lines starting from line %d (file has %d lines)", startLine, lineNumber)
	}

	return formatWithLineNumbers(lines, startLine), nil
}

func formatWithLineNumbers(lines []string, startLine int) string {
	var builder strings.Builder
	maxLineNum := startLine + len(lines) - 1
	width := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range lines {
		lineNum := startLine + i
		builder.WriteString(fmt.Sprintf("%*d: %s\n", width, lineNum, line))
	}

	return strings.TrimSuffix(builder.String(), "\n")
}

func RegisterReadFile(registry *tools.Registry, baseDir string) {
	registry.Add(&ReadFile{BaseDir: baseDir})
}
