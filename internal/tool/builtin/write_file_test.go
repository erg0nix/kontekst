package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/config"
)

func TestWriteFileExecute(t *testing.T) {
	tempDir := t.TempDir()

	existingFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &WriteFile{BaseDir: tempDir, FileConfig: config.FileToolsConfig{MaxSizeBytes: 1024}}

	tests := []struct {
		name        string
		args        map[string]any
		wantContent string
		wantErr     bool
	}{
		{
			name:        "create new file",
			args:        map[string]any{"path": "new.txt", "content": "hello world"},
			wantContent: "hello world",
		},
		{
			name:        "overwrite existing file",
			args:        map[string]any{"path": "existing.txt", "content": "updated"},
			wantContent: "updated",
		},
		{
			name:        "create file in subdirectory",
			args:        map[string]any{"path": "subdir/file.txt", "content": "nested"},
			wantContent: "nested",
		},
		{
			name:    "missing path",
			args:    map[string]any{"content": "content"},
			wantErr: true,
		},
		{
			name:    "missing content",
			args:    map[string]any{"path": "file.txt"},
			wantErr: true,
		},
		{
			name:    "absolute path rejected",
			args:    map[string]any{"path": "/tmp/bad.txt", "content": "hack"},
			wantErr: true,
		},
		{
			name:    "parent path rejected",
			args:    map[string]any{"path": "../escape.txt", "content": "hack"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(tt.args, context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %s", result)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			path := tt.args["path"].(string)
			fullPath := filepath.Join(tempDir, path)

			data, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("failed to read written file: %v", err)
				return
			}

			if string(data) != tt.wantContent {
				t.Errorf("file content = %q, want %q", string(data), tt.wantContent)
			}
		})
	}
}

func TestWriteFileMaxSize(t *testing.T) {
	tempDir := t.TempDir()
	tool := &WriteFile{BaseDir: tempDir, FileConfig: config.FileToolsConfig{MaxSizeBytes: 10}}

	_, err := tool.Execute(map[string]any{"path": "big.txt", "content": "this is way too long"}, context.Background())

	if err == nil {
		t.Error("expected error for content exceeding max size")
	}
}

func TestWriteFilePreview(t *testing.T) {
	tempDir := t.TempDir()

	existingFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &WriteFile{BaseDir: tempDir}

	tests := []struct {
		name    string
		args    map[string]any
		wantHas []string
	}{
		{
			name:    "preview new file",
			args:    map[string]any{"path": "new.txt", "content": "hello\nworld\n"},
			wantHas: []string{"--- /dev/null", "+++ new.txt", "+hello", "+world"},
		},
		{
			name:    "preview overwrite",
			args:    map[string]any{"path": "existing.txt", "content": "updated\n"},
			wantHas: []string{"--- existing.txt", "+++ existing.txt", "-original", "+updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview, err := tool.Preview(tt.args, context.Background())
			if err != nil {
				t.Fatalf("Preview error: %v", err)
			}

			for _, want := range tt.wantHas {
				if !strings.Contains(preview, want) {
					t.Errorf("preview should contain %q, got:\n%s", want, preview)
				}
			}
		})
	}
}
