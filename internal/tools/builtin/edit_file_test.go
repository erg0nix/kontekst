package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/config"
)

func TestEditFileExecute(t *testing.T) {
	tempDir := t.TempDir()
	tool := &EditFile{BaseDir: tempDir, FileConfig: config.FileToolsConfig{}}

	tests := []struct {
		name        string
		initial     string
		args        map[string]any
		wantContent string
		wantErr     bool
	}{
		{
			name:        "replace single occurrence",
			initial:     "hello world",
			args:        map[string]any{"path": "test.txt", "old_str": "world", "new_str": "universe"},
			wantContent: "hello universe",
		},
		{
			name:        "replace all occurrences",
			initial:     "foo bar foo baz foo",
			args:        map[string]any{"path": "test.txt", "old_str": "foo", "new_str": "qux", "occurrence": 0},
			wantContent: "qux bar qux baz qux",
		},
		{
			name:        "replace first occurrence only",
			initial:     "foo bar foo baz foo",
			args:        map[string]any{"path": "test.txt", "old_str": "foo", "new_str": "qux", "occurrence": 1},
			wantContent: "qux bar foo baz foo",
		},
		{
			name:        "replace second occurrence",
			initial:     "foo bar foo baz foo",
			args:        map[string]any{"path": "test.txt", "old_str": "foo", "new_str": "qux", "occurrence": 2},
			wantContent: "foo bar qux baz foo",
		},
		{
			name:        "multiline replacement",
			initial:     "line1\nold\nline3\n",
			args:        map[string]any{"path": "test.txt", "old_str": "old", "new_str": "new"},
			wantContent: "line1\nnew\nline3\n",
		},
		{
			name:    "old_str not found",
			initial: "hello world",
			args:    map[string]any{"path": "test.txt", "old_str": "notfound", "new_str": "x"},
			wantErr: true,
		},
		{
			name:    "file not found",
			initial: "",
			args:    map[string]any{"path": "nonexistent.txt", "old_str": "a", "new_str": "b"},
			wantErr: true,
		},
		{
			name:    "occurrence out of range",
			initial: "foo",
			args:    map[string]any{"path": "test.txt", "old_str": "foo", "new_str": "bar", "occurrence": 5},
			wantErr: true,
		},
		{
			name:    "absolute path rejected",
			initial: "content",
			args:    map[string]any{"path": "/etc/passwd", "old_str": "a", "new_str": "b"},
			wantErr: true,
		},
		{
			name:    "parent path rejected",
			initial: "content",
			args:    map[string]any{"path": "../escape.txt", "old_str": "a", "new_str": "b"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.initial != "" {
				testFile := filepath.Join(tempDir, "test.txt")
				if err := os.WriteFile(testFile, []byte(tt.initial), 0o644); err != nil {
					t.Fatal(err)
				}
			}

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

			testFile := filepath.Join(tempDir, "test.txt")
			data, err := os.ReadFile(testFile)
			if err != nil {
				t.Errorf("failed to read edited file: %v", err)
				return
			}

			if string(data) != tt.wantContent {
				t.Errorf("file content = %q, want %q", string(data), tt.wantContent)
			}
		})
	}
}

func TestEditFilePreview(t *testing.T) {
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &EditFile{BaseDir: tempDir}

	preview, err := tool.Preview(map[string]any{
		"path":    "test.txt",
		"old_str": "world",
		"new_str": "universe",
	}, context.Background())

	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}

	expectedParts := []string{"--- test.txt", "+++ test.txt", "-hello world", "+hello universe"}
	for _, part := range expectedParts {
		if !strings.Contains(preview, part) {
			t.Errorf("preview should contain %q, got:\n%s", part, preview)
		}
	}
}

func TestReplaceNth(t *testing.T) {
	tests := []struct {
		s     string
		old   string
		new   string
		n     int
		want  string
		wantN bool
	}{
		{"foo bar foo baz foo", "foo", "x", 1, "x bar foo baz foo", true},
		{"foo bar foo baz foo", "foo", "x", 2, "foo bar x baz foo", true},
		{"foo bar foo baz foo", "foo", "x", 3, "foo bar foo baz x", true},
		{"foo bar foo baz foo", "foo", "x", 4, "foo bar foo baz foo", false},
		{"foo", "bar", "x", 1, "foo", false},
		{"foo", "foo", "x", 0, "foo", false},
	}

	for _, tt := range tests {
		got, gotN := replaceNth(tt.s, tt.old, tt.new, tt.n)
		if got != tt.want || gotN != tt.wantN {
			t.Errorf("replaceNth(%q, %q, %q, %d) = (%q, %v), want (%q, %v)",
				tt.s, tt.old, tt.new, tt.n, got, gotN, tt.want, tt.wantN)
		}
	}
}
