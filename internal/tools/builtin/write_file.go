package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/diff"
)

const maxPreviewLines = 50

type WriteFile struct {
	BaseDir    string
	FileConfig config.FileToolsConfig
}

func (tool *WriteFile) Name() string { return "write_file" }
func (tool *WriteFile) Description() string {
	return "Creates or overwrites a file with the given content."
}
func (tool *WriteFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}
func (tool *WriteFile) RequiresApproval() bool { return true }

func (tool *WriteFile) Preview(args map[string]any, ctx context.Context) (string, error) {
	path, err := validatePath(args)
	if err != nil {
		return "", nil
	}

	content, ok := getStringArg("content", args)
	if !ok {
		return "", nil
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	existingData, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return diff.GenerateNewFileDiff(path, content, maxPreviewLines), nil
		}
		return "", nil
	}

	return diff.GenerateUnifiedDiff(path, string(existingData), content), nil
}

func (tool *WriteFile) Execute(args map[string]any, ctx context.Context) (string, error) {
	path, err := validatePath(args)
	if err != nil {
		return "", err
	}

	content, ok := getStringArg("content", args)
	if !ok {
		return "", errors.New("missing content")
	}

	if tool.FileConfig.MaxSizeBytes > 0 && int64(len(content)) > tool.FileConfig.MaxSizeBytes {
		return "", fmt.Errorf("content exceeds maximum size of %d bytes", tool.FileConfig.MaxSizeBytes)
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func RegisterWriteFile(registry *tools.Registry, baseDir string, fileConfig config.FileToolsConfig) {
	registry.Add(&WriteFile{BaseDir: baseDir, FileConfig: fileConfig})
}
