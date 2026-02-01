package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileExecute(t *testing.T) {
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &ReadFile{BaseDir: tempDir}

	tests := []struct {
		name      string
		args      map[string]any
		wantLines []string
		wantErr   bool
	}{
		{
			name:      "read full file",
			args:      map[string]any{"path": "test.txt"},
			wantLines: []string{"1: line1", "2: line2", "3: line3", "4: line4", "5: line5"},
		},
		{
			name:      "read line range",
			args:      map[string]any{"path": "test.txt", "start_line": 2, "end_line": 4},
			wantLines: []string{"2: line2", "3: line3", "4: line4"},
		},
		{
			name:      "read from start_line to end",
			args:      map[string]any{"path": "test.txt", "start_line": 3},
			wantLines: []string{"3: line3", "4: line4", "5: line5"},
		},
		{
			name:      "read single line",
			args:      map[string]any{"path": "test.txt", "start_line": 2, "end_line": 2},
			wantLines: []string{"2: line2"},
		},
		{
			name:    "missing path",
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name:    "file not found",
			args:    map[string]any{"path": "nonexistent.txt"},
			wantErr: true,
		},
		{
			name:    "absolute path rejected",
			args:    map[string]any{"path": "/etc/passwd"},
			wantErr: true,
		},
		{
			name:    "parent path rejected",
			args:    map[string]any{"path": "../secret.txt"},
			wantErr: true,
		},
		{
			name:    "invalid range - start beyond file",
			args:    map[string]any{"path": "test.txt", "start_line": 100},
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

			for _, wantLine := range tt.wantLines {
				if !strings.Contains(result, wantLine) {
					t.Errorf("result should contain %q, got:\n%s", wantLine, result)
				}
			}
		})
	}
}

func TestReadFileLineNumberWidth(t *testing.T) {
	tempDir := t.TempDir()

	var lines strings.Builder
	for i := 1; i <= 100; i++ {
		lines.WriteString("line\n")
	}

	testFile := filepath.Join(tempDir, "large.txt")
	if err := os.WriteFile(testFile, []byte(lines.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &ReadFile{BaseDir: tempDir}
	result, err := tool.Execute(map[string]any{"path": "large.txt", "start_line": 98, "end_line": 100}, context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, " 98: line") {
		t.Errorf("line numbers should be right-aligned, got:\n%s", result)
	}
	if !strings.Contains(result, "100: line") {
		t.Errorf("should contain line 100, got:\n%s", result)
	}
}

func TestFormatWithLineNumbers(t *testing.T) {
	lines := []string{"first", "second", "third"}
	result := formatWithLineNumbers(lines, 1)

	if !strings.Contains(result, "1: first") {
		t.Error("should start with line 1")
	}
	if !strings.Contains(result, "3: third") {
		t.Error("should end with line 3")
	}
}
