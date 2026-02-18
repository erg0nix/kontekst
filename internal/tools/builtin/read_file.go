package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/hashline"
)

const maxLinesDefault = 10000

type ReadFile struct {
	BaseDir string
}

func (tool *ReadFile) Name() string { return "read_file" }
func (tool *ReadFile) Description() string {
	return "Reads a file and returns its content with line numbers and hashes. Format: 'linenum:hash|content'. Supports optional line range (1-indexed, inclusive). Use the hash values when editing files with edit_file."
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
	path, err := validatePath(args)
	if err != nil {
		return "", err
	}

	startLine, _ := getIntArg("start_line", args)
	if startLine <= 0 {
		startLine = 1
	}

	endLine, hasEndLine := getIntArg("end_line", args)
	if hasEndLine && startLine > endLine {
		return "", fmt.Errorf("no lines in range %d-%d", startLine, endLine)
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var lines []string
	hashMap := make(map[int]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++

		if lineNum < startLine {
			continue
		}

		if hasEndLine && lineNum > endLine {
			break
		}

		line := scanner.Text()
		lines = append(lines, line)
		hashMap[lineNum] = hashline.ComputeLineHash(line)

		if !hasEndLine && len(lines) >= maxLinesDefault {
			return "", fmt.Errorf("file has more than %d lines; specify start_line and end_line to read a range", maxLinesDefault)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan file: %w", err)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no lines starting from line %d (file has %d lines)", startLine, lineNum)
	}

	return formatWithLineNumbers(lines, startLine, hashMap), nil
}

func formatWithLineNumbers(lines []string, startLine int, hashMap map[int]string) string {
	var builder strings.Builder

	maxLineNum := startLine + len(lines) - 1
	width := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range lines {
		lineNum := startLine + i
		hash := hashMap[lineNum]
		builder.WriteString(fmt.Sprintf("%*d:%s|%s\n", width, lineNum, hash, line))
	}

	return strings.TrimSuffix(builder.String(), "\n")
}

func RegisterReadFile(registry *tools.Registry, baseDir string) {
	registry.Add(&ReadFile{BaseDir: baseDir})
}
