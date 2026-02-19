package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFilesExecute(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "file.go"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "subdir", "nested.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &ListFiles{BaseDir: tempDir}

	tests := []struct {
		name        string
		args        map[string]any
		wantFiles   []string
		wantMissing []string
		wantErr     bool
	}{
		{
			name:      "list all txt files",
			args:      map[string]any{"pattern": "*.txt"},
			wantFiles: []string{"file1.txt", "file2.txt"},
		},
		{
			name:      "list go files",
			args:      map[string]any{"pattern": "*.go"},
			wantFiles: []string{"file.go"},
		},
		{
			name:        "no matches",
			args:        map[string]any{"pattern": "*.rs"},
			wantMissing: []string{"rs"},
		},
		{
			name:    "missing pattern",
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name:    "absolute path rejected",
			args:    map[string]any{"pattern": "/etc/*"},
			wantErr: true,
		},
		{
			name:    "parent path rejected",
			args:    map[string]any{"pattern": "../*"},
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

			for _, want := range tt.wantFiles {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got:\n%s", want, result)
				}
			}

			for _, missing := range tt.wantMissing {
				if strings.Contains(result, missing) {
					t.Errorf("result should not contain %q, got:\n%s", missing, result)
				}
			}
		})
	}
}

func TestListFilesExcludesDirectories(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tempDir, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "afile"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &ListFiles{BaseDir: tempDir}
	result, err := tool.Execute(map[string]any{"pattern": "a*"}, context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result, "adir") {
		t.Error("directories should be excluded from results")
	}
	if !strings.Contains(result, "afile") {
		t.Error("files should be included in results")
	}
}
