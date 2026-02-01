package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/tools"
)

type EditFile struct {
	BaseDir    string
	FileConfig config.FileToolsConfig
}

func (tool *EditFile) Name() string { return "edit_file" }
func (tool *EditFile) Description() string {
	return "Edits a file by replacing occurrences of old_str with new_str. File must exist and old_str must be found."
}
func (tool *EditFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file to edit",
			},
			"old_str": map[string]any{
				"type":        "string",
				"description": "The text to find and replace",
			},
			"new_str": map[string]any{
				"type":        "string",
				"description": "The replacement text",
			},
			"occurrence": map[string]any{
				"type":        "integer",
				"description": "Which occurrence to replace: 0 for all, 1 for first, 2 for second, etc. (default: 0)",
			},
		},
		"required": []string{"path", "old_str", "new_str"},
	}
}
func (tool *EditFile) RequiresApproval() bool { return true }

func (tool *EditFile) Preview(args map[string]any, ctx context.Context) (string, error) {
	path, ok := getStringArg("path", args)
	if !ok || path == "" {
		return "", nil
	}

	if !isSafeRelative(path) {
		return "", nil
	}

	oldStr, ok := getStringArg("old_str", args)
	if !ok {
		return "", nil
	}

	newStr, ok := getStringArg("new_str", args)
	if !ok {
		return "", nil
	}

	occurrence, _ := getIntArg("occurrence", args)

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", nil
	}

	content := string(data)
	if !strings.Contains(content, oldStr) {
		return "", nil
	}

	var newContent string
	if occurrence == 0 {
		newContent = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		newContent, _ = replaceNth(content, oldStr, newStr, occurrence)
	}

	return generateUnifiedDiff(path, content, newContent), nil
}

func (tool *EditFile) Execute(args map[string]any, ctx context.Context) (string, error) {
	path, ok := getStringArg("path", args)
	if !ok || path == "" {
		return "", errors.New("missing path")
	}

	if !isSafeRelative(path) {
		return "", errors.New("absolute or parent paths are not allowed")
	}

	oldStr, ok := getStringArg("old_str", args)
	if !ok {
		return "", errors.New("missing old_str")
	}

	newStr, ok := getStringArg("new_str", args)
	if !ok {
		return "", errors.New("missing new_str")
	}

	occurrence, _ := getIntArg("occurrence", args)

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", err
	}

	content := string(data)

	if !strings.Contains(content, oldStr) {
		return "", errors.New("old_str not found in file")
	}

	var newContent string
	replacementCount := 0

	if occurrence == 0 {
		newContent = strings.ReplaceAll(content, oldStr, newStr)
		replacementCount = strings.Count(content, oldStr)
	} else {
		newContent, replacementCount = replaceNth(content, oldStr, newStr, occurrence)
		if replacementCount == 0 {
			return "", fmt.Errorf("occurrence %d of old_str not found (only %d occurrences exist)", occurrence, strings.Count(content, oldStr))
		}
	}

	if tool.FileConfig.MaxSizeBytes > 0 && int64(len(newContent)) > tool.FileConfig.MaxSizeBytes {
		return "", fmt.Errorf("result would exceed maximum file size of %d bytes", tool.FileConfig.MaxSizeBytes)
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return "", err
	}

	if occurrence == 0 {
		return fmt.Sprintf("Replaced %d occurrence(s) in %s", replacementCount, path), nil
	}
	return fmt.Sprintf("Replaced occurrence %d in %s", occurrence, path), nil
}

func replaceNth(s, old, new string, n int) (string, int) {
	if n <= 0 {
		return s, 0
	}

	index := 0
	for i := 1; i <= n; i++ {
		pos := strings.Index(s[index:], old)
		if pos == -1 {
			return s, 0
		}
		if i == n {
			return s[:index+pos] + new + s[index+pos+len(old):], 1
		}
		index += pos + len(old)
	}

	return s, 0
}

func RegisterEditFile(registry *tools.Registry, baseDir string, fileConfig config.FileToolsConfig) {
	registry.Add(&EditFile{BaseDir: baseDir, FileConfig: fileConfig})
}
