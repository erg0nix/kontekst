package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/erg0nix/kontekst/internal/config"
	toolpkg "github.com/erg0nix/kontekst/internal/tool"
	tooldiff "github.com/erg0nix/kontekst/internal/tool/diff"
	"github.com/erg0nix/kontekst/internal/tool/hashline"
)

// EditFile is a tool that applies hashline-validated edits to existing files.
type EditFile struct {
	BaseDir    string
	FileConfig config.FileToolsConfig
}

type edit struct {
	operation string
	line      int
	hash      string
	content   string
}

type editPlan struct {
	path            string
	fullPath        string
	oldLines        []string
	newLines        []string
	edits           []edit
	trailingNewline bool
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

func (tool *EditFile) prepareEdits(args map[string]any, ctx context.Context) (*editPlan, error) {
	path, err := validatePath(args)
	if err != nil {
		return nil, err
	}

	editsRaw, ok := args["edits"]
	if !ok {
		return nil, errors.New("missing edits parameter")
	}

	edits, err := parseEdits(editsRaw)
	if err != nil {
		return nil, err
	}

	if len(edits) == 0 {
		return nil, errors.New("edits array is empty")
	}

	baseDir := resolveBaseDir(ctx, tool.BaseDir)
	fullPath := filepath.Join(baseDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", path)
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	trailingNewline := len(data) > 0 && data[len(data)-1] == '\n'
	lines := tooldiff.SplitLines(string(data))

	if err := validateEdits(edits, lines); err != nil {
		return nil, err
	}

	newLines, err := applyEdits(lines, edits)
	if err != nil {
		return nil, err
	}

	return &editPlan{
		path:            path,
		fullPath:        fullPath,
		oldLines:        lines,
		newLines:        newLines,
		edits:           edits,
		trailingNewline: trailingNewline,
	}, nil
}

func assembleContent(lines []string, trailingNewline bool) string {
	content := strings.Join(lines, "\n")
	if trailingNewline && len(lines) > 0 {
		content += "\n"
	}
	return content
}

func (tool *EditFile) Preview(args map[string]any, ctx context.Context) (string, error) {
	preview, err := tool.PreviewStructured(args, ctx)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(preview)
	if err != nil {
		return "", fmt.Errorf("edit_file: marshal preview: %w", err)
	}

	return string(data), nil
}

// PreviewStructured returns a structured diff preview of the edits that would be applied.
func (tool *EditFile) PreviewStructured(args map[string]any, ctx context.Context) (*tooldiff.DiffPreview, error) {
	plan, err := tool.prepareEdits(args, ctx)
	if err != nil {
		return nil, err
	}

	oldContent := assembleContent(plan.oldLines, plan.trailingNewline)
	newContent := assembleContent(plan.newLines, plan.trailingNewline)

	oldHashes := hashline.GenerateHashMap(plan.oldLines)
	newHashes := hashline.GenerateHashMap(plan.newLines)

	preview := tooldiff.GenerateStructuredDiffWithHashes(plan.path, oldContent, newContent, oldHashes, newHashes)

	opCounts := make(map[string]int)
	for _, e := range plan.edits {
		opCounts[e.operation]++
	}
	preview.Summary.Operations = opCounts
	preview.Summary.TotalEdits = len(plan.edits)

	return &preview, nil
}

func (tool *EditFile) Execute(args map[string]any, ctx context.Context) (string, error) {
	plan, err := tool.prepareEdits(args, ctx)
	if err != nil {
		return "", err
	}

	newContent := assembleContent(plan.newLines, plan.trailingNewline)

	if tool.FileConfig.MaxSizeBytes > 0 && int64(len(newContent)) > tool.FileConfig.MaxSizeBytes {
		return "", fmt.Errorf("result would exceed maximum file size of %d bytes", tool.FileConfig.MaxSizeBytes)
	}

	if err := os.WriteFile(plan.fullPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("Applied %d edit(s) to %s", len(plan.edits), plan.path), nil
}

func parseEdits(editsRaw any) ([]edit, error) {
	editsSlice, ok := editsRaw.([]any)
	if !ok {
		return nil, errors.New("edits must be an array")
	}

	var edits []edit
	for i, editRaw := range editsSlice {
		editMap, ok := editRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("edit %d is not an object", i)
		}

		op, ok := editMap["operation"].(string)
		if !ok {
			return nil, fmt.Errorf("edit %d missing operation", i)
		}

		switch op {
		case "replace", "insert_after", "insert_before", "delete":
		default:
			return nil, fmt.Errorf("edit %d: invalid operation %q", i, op)
		}

		lineFloat, ok := editMap["line"].(float64)
		if !ok {
			return nil, fmt.Errorf("edit %d missing line number", i)
		}

		hash, ok := editMap["hash"].(string)
		if !ok {
			return nil, fmt.Errorf("edit %d missing hash", i)
		}

		content := ""
		if op != "delete" {
			content, ok = editMap["content"].(string)
			if !ok {
				return nil, fmt.Errorf("edit %d missing content for operation %s", i, op)
			}
		}

		edits = append(edits, edit{
			operation: op,
			line:      int(lineFloat),
			hash:      hash,
			content:   content,
		})
	}

	return edits, nil
}

func validateEdits(edits []edit, lines []string) error {
	hashMap := hashline.GenerateHashMap(lines)

	seen := make(map[int]bool, len(edits))
	for i, e := range edits {
		if seen[e.line] {
			return fmt.Errorf("edit %d: duplicate edit on line %d", i, e.line)
		}
		seen[e.line] = true

		if e.line < 1 || e.line > len(lines) {
			return fmt.Errorf("edit %d: line %d out of range (file has %d lines)", i, e.line, len(lines))
		}

		actualHash := hashMap[e.line]
		if actualHash != e.hash {
			return fmt.Errorf(
				"edit %d: hash mismatch on line %d: expected %s, found %s (content: %q)",
				i, e.line, e.hash, actualHash, lines[e.line-1])
		}
	}

	return nil
}

func applyEdits(lines []string, edits []edit) ([]string, error) {
	sorted := make([]edit, len(edits))
	copy(sorted, edits)
	slices.SortFunc(sorted, func(a, b edit) int { return b.line - a.line })

	result := make([]string, len(lines))
	copy(result, lines)

	for _, e := range sorted {
		switch e.operation {
		case "replace":
			result[e.line-1] = e.content
		case "delete":
			result = slices.Delete(result, e.line-1, e.line)
		case "insert_after":
			result = slices.Insert(result, e.line, e.content)
		case "insert_before":
			result = slices.Insert(result, e.line-1, e.content)
		default:
			return nil, fmt.Errorf("unknown operation: %s", e.operation)
		}
	}

	return result, nil
}

// RegisterEditFile adds the edit_file tool to the registry.
func RegisterEditFile(registry *toolpkg.Registry, baseDir string, fileConfig config.FileToolsConfig) {
	registry.Add(&EditFile{BaseDir: baseDir, FileConfig: fileConfig})
}
