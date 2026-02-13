package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/tools"
)

type EditFile struct {
	BaseDir    string
	FileConfig config.FileToolsConfig
}

type Edit struct {
	Operation string `json:"operation"`
	Line      int    `json:"line"`
	Hash      string `json:"hash"`
	Content   string `json:"content,omitempty"`
}

func (tool *EditFile) Name() string { return "edit_file" }

func (tool *EditFile) Description() string {
	return "Edits a file using hashline references. Operations: 'replace' (replace line content), 'insert_after' (insert new line after specified line), 'insert_before' (insert new line before specified line), 'delete' (delete line). All operations validate the hash before applying changes. Use hashes from read_file output."
}

func (tool *EditFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file to edit",
			},
			"edits": map[string]any{
				"type":        "array",
				"description": "Array of edit operations to apply atomically",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"operation": map[string]any{
							"type":        "string",
							"description": "Edit operation: 'replace', 'insert_after', 'insert_before', or 'delete'",
							"enum":        []string{"replace", "insert_after", "insert_before", "delete"},
						},
						"line": map[string]any{
							"type":        "integer",
							"description": "Line number (1-indexed) to edit or use as anchor",
						},
						"hash": map[string]any{
							"type":        "string",
							"description": "Hash of the line for validation (from read_file output)",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "New content (required for replace/insert operations, omitted for delete)",
						},
					},
					"required": []string{"operation", "line", "hash"},
				},
			},
		},
		"required": []string{"path", "edits"},
	}
}

func (tool *EditFile) RequiresApproval() bool { return true }

func (tool *EditFile) Preview(args map[string]any, ctx context.Context) (string, error) {
	path, err := validatePath(args)
	if err != nil {
		return "", nil
	}

	editsRaw, ok := args["edits"]
	if !ok {
		return "", nil
	}

	edits, err := parseEdits(editsRaw)
	if err != nil {
		return "", nil
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", nil
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if err := validateEdits(edits, lines); err != nil {
		return "", nil
	}

	newLines, err := applyEdits(lines, edits)
	if err != nil {
		return "", nil
	}

	oldContent := string(data)
	newContent := strings.Join(newLines, "\n")
	if len(newLines) > 0 {
		newContent += "\n"
	}

	return generateUnifiedDiff(path, oldContent, newContent), nil
}

func (tool *EditFile) Execute(args map[string]any, ctx context.Context) (string, error) {
	path, err := validatePath(args)
	if err != nil {
		return "", err
	}

	editsRaw, ok := args["edits"]
	if !ok {
		return "", errors.New("missing edits parameter")
	}

	edits, err := parseEdits(editsRaw)
	if err != nil {
		return "", fmt.Errorf("failed to parse edits: %w", err)
	}

	if len(edits) == 0 {
		return "", errors.New("edits array is empty")
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if err := validateEdits(edits, lines); err != nil {
		return "", err
	}

	newLines, err := applyEdits(lines, edits)
	if err != nil {
		return "", err
	}

	newContent := strings.Join(newLines, "\n")
	if len(newLines) > 0 {
		newContent += "\n"
	}

	if tool.FileConfig.MaxSizeBytes > 0 && int64(len(newContent)) > tool.FileConfig.MaxSizeBytes {
		return "", fmt.Errorf("result would exceed maximum file size of %d bytes", tool.FileConfig.MaxSizeBytes)
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Applied %d edit(s) to %s", len(edits), path), nil
}

func parseEdits(editsRaw any) ([]Edit, error) {
	editsSlice, ok := editsRaw.([]any)
	if !ok {
		return nil, errors.New("edits must be an array")
	}

	var edits []Edit
	for i, editRaw := range editsSlice {
		editMap, ok := editRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("edit %d is not an object", i)
		}

		operation, ok := editMap["operation"].(string)
		if !ok {
			return nil, fmt.Errorf("edit %d missing operation", i)
		}

		lineFloat, ok := editMap["line"].(float64)
		if !ok {
			return nil, fmt.Errorf("edit %d missing line number", i)
		}
		line := int(lineFloat)

		hash, ok := editMap["hash"].(string)
		if !ok {
			return nil, fmt.Errorf("edit %d missing hash", i)
		}

		content := ""
		if operation != "delete" {
			content, ok = editMap["content"].(string)
			if !ok {
				return nil, fmt.Errorf("edit %d missing content for operation %s", i, operation)
			}
		}

		edits = append(edits, Edit{
			Operation: operation,
			Line:      line,
			Hash:      hash,
			Content:   content,
		})
	}

	return edits, nil
}

func validateEdits(edits []Edit, lines []string) error {
	for i, edit := range edits {
		if edit.Line < 1 || edit.Line > len(lines) {
			return fmt.Errorf("edit %d: line %d out of range (file has %d lines)", i, edit.Line, len(lines))
		}

		actualLine := lines[edit.Line-1]
		actualHash := computeLineHash(actualLine)

		if actualHash != edit.Hash {
			return fmt.Errorf(
				"edit %d: hash mismatch on line %d: expected %s, found %s\n"+
					"Actual content: %q\n"+
					"File may have been modified since read",
				i, edit.Line, edit.Hash, actualHash, actualLine)
		}

		validOps := map[string]bool{"replace": true, "insert_after": true, "insert_before": true, "delete": true}
		if !validOps[edit.Operation] {
			return fmt.Errorf("edit %d: invalid operation %q", i, edit.Operation)
		}

		if edit.Operation != "delete" && edit.Content == "" {
			return fmt.Errorf("edit %d: content required for operation %s", i, edit.Operation)
		}
	}

	return nil
}

func applyEdits(lines []string, edits []Edit) ([]string, error) {
	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		return sortedEdits[i].Line > sortedEdits[j].Line
	})

	result := make([]string, len(lines))
	copy(result, lines)

	for _, edit := range sortedEdits {
		switch edit.Operation {
		case "replace":
			result[edit.Line-1] = edit.Content

		case "delete":
			result = append(result[:edit.Line-1], result[edit.Line:]...)

		case "insert_after":
			newResult := make([]string, 0, len(result)+1)
			newResult = append(newResult, result[:edit.Line]...)
			newResult = append(newResult, edit.Content)
			newResult = append(newResult, result[edit.Line:]...)
			result = newResult

		case "insert_before":
			newResult := make([]string, 0, len(result)+1)
			newResult = append(newResult, result[:edit.Line-1]...)
			newResult = append(newResult, edit.Content)
			newResult = append(newResult, result[edit.Line-1:]...)
			result = newResult

		default:
			return nil, fmt.Errorf("unknown operation: %s", edit.Operation)
		}
	}

	return result, nil
}

func RegisterEditFile(registry *tools.Registry, baseDir string, fileConfig config.FileToolsConfig) {
	registry.Add(&EditFile{BaseDir: baseDir, FileConfig: fileConfig})
}
