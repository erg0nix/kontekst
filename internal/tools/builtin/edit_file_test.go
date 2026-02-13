package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/tools/diff"
	"github.com/erg0nix/kontekst/internal/tools/hashline"
)

func TestEditFileReplace(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "modified line2",
			},
		},
	}

	result, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result, "Applied 1 edit") {
		t.Errorf("unexpected result: %s", result)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nmodified line2\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileInsertAfter(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "insert_after",
				"line":      float64(2),
				"hash":      hash,
				"content":   "inserted line",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nline2\ninserted line\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileInsertBefore(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "insert_before",
				"line":      float64(2),
				"hash":      hash,
				"content":   "inserted line",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\ninserted line\nline2\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileDelete(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "delete",
				"line":      float64(2),
				"hash":      hash,
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileMultipleEdits(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\nline4\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash2 := hashline.ComputeLineHash("line2")
	hash4 := hashline.ComputeLineHash("line4")

	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash2,
				"content":   "modified line2",
			},
			map[string]any{
				"operation": "delete",
				"line":      float64(4),
				"hash":      hash4,
			},
		},
	}

	result, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result, "Applied 2 edit") {
		t.Errorf("unexpected result: %s", result)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nmodified line2\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileHashMismatch(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      "xxx",
				"content":   "modified line2",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}

	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error should mention hash mismatch, got: %v", err)
	}

	if !strings.Contains(err.Error(), "line2") {
		t.Errorf("error should show actual content, got: %v", err)
	}
}

func TestEditFileLineOutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(100),
				"hash":      "xxx",
				"content":   "new content",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for line out of range")
	}

	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range, got: %v", err)
	}
}

func TestEditFileInvalidOperation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line1")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "invalid_op",
				"line":      float64(1),
				"hash":      hash,
				"content":   "new content",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for invalid operation")
	}

	if !strings.Contains(err.Error(), "invalid operation") {
		t.Errorf("error should mention invalid operation, got: %v", err)
	}
}

func TestEditFileEmptyEdits(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	args := map[string]any{
		"path":  "test.txt",
		"edits": []any{},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for empty edits array")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty edits, got: %v", err)
	}
}

func TestEditFileMissingContent(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line1")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(1),
				"hash":      hash,
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for missing content")
	}

	if !strings.Contains(err.Error(), "missing content") {
		t.Errorf("error should mention missing content, got: %v", err)
	}
}

func TestEditFileMaxSize(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{
		BaseDir: tempDir,
		FileConfig: config.FileToolsConfig{
			MaxSizeBytes: 5,
		},
	}

	hash := hashline.ComputeLineHash("line1")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(1),
				"hash":      hash,
				"content":   "very long content that exceeds limit",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for exceeding max size")
	}

	if !strings.Contains(err.Error(), "exceed maximum file size") {
		t.Errorf("error should mention max size, got: %v", err)
	}
}

func TestEditFilePreview(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "modified line2",
			},
		},
	}

	preview, err := tool.Preview(args, context.Background())
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	var diffPreview diff.DiffPreview
	if err := json.Unmarshal([]byte(preview), &diffPreview); err != nil {
		t.Fatalf("failed to unmarshal preview: %v", err)
	}

	if len(diffPreview.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(diffPreview.Blocks))
	}

	hasDeleteLine := false
	hasInsertLine := false
	for _, line := range diffPreview.Blocks[0].Lines {
		if line.Type == "delete" && strings.Contains(line.Content, "line2") {
			hasDeleteLine = true
		}
		if line.Type == "insert" && strings.Contains(line.Content, "modified line2") {
			hasInsertLine = true
		}
	}

	if !hasDeleteLine {
		t.Error("preview should show deleted line2")
	}
	if !hasInsertLine {
		t.Error("preview should show inserted modified line2")
	}
}

func TestEditFileSorting(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash2 := hashline.ComputeLineHash("line2")
	hash4 := hashline.ComputeLineHash("line4")

	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "insert_after",
				"line":      float64(2),
				"hash":      hash2,
				"content":   "inserted after 2",
			},
			map[string]any{
				"operation": "insert_after",
				"line":      float64(4),
				"hash":      hash4,
				"content":   "inserted after 4",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nline2\ninserted after 2\nline3\nline4\ninserted after 4\nline5\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileCollisionHandling(t *testing.T) {
	tempDir := t.TempDir()
	tool := &EditFile{BaseDir: tempDir}
	lines := []string{"dup", "dup", "unique"}

	hashMap, _ := hashline.GenerateHashMap(lines)
	hash1 := hashMap[1]
	hash2 := hashMap[2]

	if hash1 == hash2 {
		t.Fatal("collision detection should produce different hashes")
	}

	t.Run("edit first duplicate", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "first.txt")
		if err := os.WriteFile(testFile, []byte("dup\ndup\nunique\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		args := map[string]any{
			"path": "first.txt",
			"edits": []any{
				map[string]any{
					"operation": "replace",
					"line":      float64(1),
					"hash":      hash1,
					"content":   "modified first dup",
				},
			},
		}

		_, err := tool.Execute(args, context.Background())
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		got, _ := os.ReadFile(testFile)
		if string(got) != "modified first dup\ndup\nunique\n" {
			t.Errorf("content = %q, want %q", got, "modified first dup\ndup\nunique\n")
		}
	})

	t.Run("edit second duplicate", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "second.txt")
		if err := os.WriteFile(testFile, []byte("dup\ndup\nunique\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		args := map[string]any{
			"path": "second.txt",
			"edits": []any{
				map[string]any{
					"operation": "replace",
					"line":      float64(2),
					"hash":      hash2,
					"content":   "modified second dup",
				},
			},
		}

		_, err := tool.Execute(args, context.Background())
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		got, _ := os.ReadFile(testFile)
		if string(got) != "dup\nmodified second dup\nunique\n" {
			t.Errorf("content = %q, want %q", got, "dup\nmodified second dup\nunique\n")
		}
	})
}

func TestEditFilePreviewStructured(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "modified",
			},
		},
	}

	preview, err := tool.PreviewStructured(args, context.Background())
	if err != nil {
		t.Fatalf("PreviewStructured failed: %v", err)
	}

	if preview == nil {
		t.Fatal("preview is nil")
	}

	if preview.Path != "test.txt" {
		t.Errorf("path = %q, want %q", preview.Path, "test.txt")
	}

	if len(preview.Blocks) != 1 {
		t.Errorf("blocks count = %d, want 1", len(preview.Blocks))
	}

	if preview.Summary.LinesAdded != 1 {
		t.Errorf("lines added = %d, want 1", preview.Summary.LinesAdded)
	}

	if preview.Summary.LinesRemoved != 1 {
		t.Errorf("lines removed = %d, want 1", preview.Summary.LinesRemoved)
	}

	if preview.Summary.Operations["replace"] != 1 {
		t.Errorf("replace operations = %d, want 1", preview.Summary.Operations["replace"])
	}
}

func TestEditFilePreviewWithHashes(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "modified",
			},
		},
	}

	preview, err := tool.PreviewStructured(args, context.Background())
	if err != nil {
		t.Fatalf("PreviewStructured failed: %v", err)
	}

	if len(preview.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(preview.Blocks))
	}

	block := preview.Blocks[0]
	hasHashedLine := false
	for _, line := range block.Lines {
		if line.Hash != nil {
			hasHashedLine = true
			break
		}
	}

	if !hasHashedLine {
		t.Error("expected at least one line with hash annotation")
	}
}

func TestEditFilePreviewJSON(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "modified",
			},
		},
	}

	previewStr, err := tool.Preview(args, context.Background())
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	var preview diff.DiffPreview
	if err := json.Unmarshal([]byte(previewStr), &preview); err != nil {
		t.Fatalf("failed to unmarshal preview JSON: %v", err)
	}

	if preview.Path != "test.txt" {
		t.Errorf("path = %q, want %q", preview.Path, "test.txt")
	}

	if len(preview.Blocks) != 1 {
		t.Errorf("blocks count = %d, want 1", len(preview.Blocks))
	}
}

func TestEditFilePreviewMultipleEdits(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\nline4\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash2 := hashline.ComputeLineHash("line2")
	hash3 := hashline.ComputeLineHash("line3")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash2,
				"content":   "modified2",
			},
			map[string]any{
				"operation": "delete",
				"line":      float64(3),
				"hash":      hash3,
			},
		},
	}

	preview, err := tool.PreviewStructured(args, context.Background())
	if err != nil {
		t.Fatalf("PreviewStructured failed: %v", err)
	}

	if preview.Summary.Operations["replace"] != 1 {
		t.Errorf("replace operations = %d, want 1", preview.Summary.Operations["replace"])
	}

	if preview.Summary.Operations["delete"] != 1 {
		t.Errorf("delete operations = %d, want 1", preview.Summary.Operations["delete"])
	}

	if preview.Summary.TotalEdits != 2 {
		t.Errorf("total edits = %d, want 2", preview.Summary.TotalEdits)
	}
}

func TestEditFileDuplicateLineTarget(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line2")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "first edit",
			},
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      hash,
				"content":   "second edit",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err == nil {
		t.Fatal("expected error for duplicate line target")
	}

	if !strings.Contains(err.Error(), "duplicate edit on line 2") {
		t.Errorf("error should mention duplicate edit, got: %v", err)
	}
}

func TestEditFileNoTrailingNewline(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := hashline.ComputeLineHash("line1")
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(1),
				"hash":      hash,
				"content":   "modified",
			},
		},
	}

	_, err := tool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "modified\nline2\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}

func TestEditFileRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	readTool := &ReadFile{BaseDir: tempDir}
	readResult, err := readTool.Execute(map[string]any{"path": "test.txt"}, context.Background())
	if err != nil {
		t.Fatalf("read_file failed: %v", err)
	}

	var line2Hash string
	for _, outputLine := range strings.Split(readResult, "\n") {
		if strings.Contains(outputLine, "|line2") {
			parts := strings.SplitN(outputLine, ":", 2)
			if len(parts) == 2 {
				hashAndContent := strings.SplitN(parts[1], "|", 2)
				line2Hash = hashAndContent[0]
			}
		}
	}

	if line2Hash == "" {
		t.Fatalf("could not extract hash for line2 from read output: %s", readResult)
	}

	editTool := &EditFile{BaseDir: tempDir}
	args := map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{
				"operation": "replace",
				"line":      float64(2),
				"hash":      line2Hash,
				"content":   "modified line2",
			},
		},
	}

	_, err = editTool.Execute(args, context.Background())
	if err != nil {
		t.Fatalf("edit_file failed: %v", err)
	}

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line1\nmodified line2\nline3\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}
