package builtin

import (
	"strings"
	"testing"
)

func TestGenerateUnifiedDiff(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		oldContent string
		newContent string
		wantHas    []string
		wantEmpty  bool
	}{
		{
			name:       "single line change",
			path:       "test.txt",
			oldContent: "line1\nline2\nline3\n",
			newContent: "line1\nmodified\nline3\n",
			wantHas:    []string{"--- test.txt", "+++ test.txt", "-line2", "+modified"},
		},
		{
			name:       "add line",
			path:       "test.txt",
			oldContent: "line1\nline2\n",
			newContent: "line1\nline2\nline3\n",
			wantHas:    []string{"+line3"},
		},
		{
			name:       "remove line",
			path:       "test.txt",
			oldContent: "line1\nline2\nline3\n",
			newContent: "line1\nline3\n",
			wantHas:    []string{"-line2"},
		},
		{
			name:       "no change",
			path:       "test.txt",
			oldContent: "same\n",
			newContent: "same\n",
			wantEmpty:  true,
		},
		{
			name:       "empty to content",
			path:       "test.txt",
			oldContent: "",
			newContent: "new content\n",
			wantHas:    []string{"+new content"},
		},
		{
			name:       "multiline change",
			path:       "code.go",
			oldContent: "func main() {\n\tfmt.Println(\"hello\")\n}\n",
			newContent: "func main() {\n\tfmt.Println(\"world\")\n}\n",
			wantHas:    []string{"-\tfmt.Println(\"hello\")", "+\tfmt.Println(\"world\")"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateUnifiedDiff(tt.path, tt.oldContent, tt.newContent)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty diff, got:\n%s", got)
				}
				return
			}

			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("diff should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestGenerateNewFileDiff(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  string
		maxLines int
		wantHas  []string
	}{
		{
			name:     "small file",
			path:     "new.txt",
			content:  "line1\nline2\nline3\n",
			maxLines: 50,
			wantHas:  []string{"--- /dev/null", "+++ new.txt", "+line1", "+line2", "+line3"},
		},
		{
			name:     "truncated file",
			path:     "large.txt",
			content:  "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
			maxLines: 5,
			wantHas:  []string{"--- /dev/null", "+++ large.txt", "omitted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNewFileDiff(tt.path, tt.content, tt.maxLines)

			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("diff should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"a\n", 1},
		{"a\nb", 2},
		{"a\nb\n", 2},
		{"a\nb\nc", 3},
	}

	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}
