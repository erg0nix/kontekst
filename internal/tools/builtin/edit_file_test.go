package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/config"
)

func TestEditFileReplace(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hash := computeLineHash("line2")
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

	hash := computeLineHash("line2")
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

	hash := computeLineHash("line2")
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

	hash := computeLineHash("line2")
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

	hash2 := computeLineHash("line2")
	hash4 := computeLineHash("line4")

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

	hash := computeLineHash("line1")
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

	hash := computeLineHash("line1")
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

	hash := computeLineHash("line1")
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

	hash := computeLineHash("line2")
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

	if !strings.Contains(preview, "-line2") {
		t.Errorf("preview should show removed line, got: %s", preview)
	}
	if !strings.Contains(preview, "+modified line2") {
		t.Errorf("preview should show added line, got: %s", preview)
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

	hash2 := computeLineHash("line2")
	hash4 := computeLineHash("line4")

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
	testFile := filepath.Join(tempDir, "test.txt")
	content := "dup\ndup\nunique\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	hashMap, _ := generateHashMap([]string{"dup", "dup", "unique"})
	hash1 := hashMap[1]
	hash2 := hashMap[2]

	if hash1 == hash2 {
		t.Fatal("collision detection should produce different hashes")
	}

	args := map[string]any{
		"path": "test.txt",
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

	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "modified first dup\ndup\nunique\n"
	if string(newContent) != expected {
		t.Errorf("content = %q, want %q", string(newContent), expected)
	}
}
