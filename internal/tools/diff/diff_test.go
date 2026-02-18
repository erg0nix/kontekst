package diff

import (
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/tools/hashline"
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
			got := GenerateUnifiedDiff(tt.path, tt.oldContent, tt.newContent)

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
			got := GenerateNewFileDiff(tt.path, tt.content, tt.maxLines)

			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("diff should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestGenerateStructuredDiff(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		oldContent  string
		newContent  string
		wantBlocks  int
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "single line change",
			path:        "test.txt",
			oldContent:  "line1\nline2\nline3\n",
			newContent:  "line1\nmodified\nline3\n",
			wantBlocks:  1,
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "add line",
			path:        "test.txt",
			oldContent:  "line1\nline2\n",
			newContent:  "line1\nline2\nline3\n",
			wantBlocks:  1,
			wantAdded:   1,
			wantRemoved: 0,
		},
		{
			name:        "remove line",
			path:        "test.txt",
			oldContent:  "line1\nline2\nline3\n",
			newContent:  "line1\nline3\n",
			wantBlocks:  1,
			wantAdded:   0,
			wantRemoved: 1,
		},
		{
			name:        "no change",
			path:        "test.txt",
			oldContent:  "same\n",
			newContent:  "same\n",
			wantBlocks:  0,
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "empty to content",
			path:        "test.txt",
			oldContent:  "",
			newContent:  "new content\n",
			wantBlocks:  1,
			wantAdded:   1,
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateStructuredDiff(tt.path, tt.oldContent, tt.newContent)

			if got.Path != tt.path {
				t.Errorf("path = %q, want %q", got.Path, tt.path)
			}

			if len(got.Blocks) != tt.wantBlocks {
				t.Errorf("blocks count = %d, want %d", len(got.Blocks), tt.wantBlocks)
			}

			if got.Summary.LinesAdded != tt.wantAdded {
				t.Errorf("lines added = %d, want %d", got.Summary.LinesAdded, tt.wantAdded)
			}

			if got.Summary.LinesRemoved != tt.wantRemoved {
				t.Errorf("lines removed = %d, want %d", got.Summary.LinesRemoved, tt.wantRemoved)
			}

			wantNetChange := tt.wantAdded - tt.wantRemoved
			if got.Summary.NetChange != wantNetChange {
				t.Errorf("net change = %d, want %d", got.Summary.NetChange, wantNetChange)
			}
		})
	}
}

func TestGenerateStructuredDiffWithHashes(t *testing.T) {
	oldContent := "line1\nline2\nline3\n"
	newContent := "line1\nmodified\nline3\n"

	oldLines := SplitLines(oldContent)
	newLines := SplitLines(newContent)

	oldHashes := make(map[int]string)
	for i, line := range oldLines {
		oldHashes[i+1] = hashline.ComputeLineHash(line)
	}

	newHashes := make(map[int]string)
	for i, line := range newLines {
		newHashes[i+1] = hashline.ComputeLineHash(line)
	}

	diff := GenerateStructuredDiffWithHashes("test.txt", oldContent, newContent, oldHashes, newHashes)

	if len(diff.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(diff.Blocks))
	}

	block := diff.Blocks[0]
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

	contextLines := 0
	insertLines := 0
	deleteLines := 0
	for _, line := range block.Lines {
		switch line.Type {
		case "context":
			contextLines++
			if line.Hash == nil {
				t.Error("context line missing hash")
			}
		case "insert":
			insertLines++
		case "delete":
			deleteLines++
			if line.Hash == nil {
				t.Error("delete line missing hash")
			}
		}
	}

	if insertLines != 1 {
		t.Errorf("expected 1 insert line, got %d", insertLines)
	}

	if deleteLines != 1 {
		t.Errorf("expected 1 delete line, got %d", deleteLines)
	}
}

func TestStructuredDiffSummary(t *testing.T) {
	tests := []struct {
		name        string
		oldContent  string
		newContent  string
		wantAdded   int
		wantRemoved int
		wantNet     int
	}{
		{
			name:        "balanced change",
			oldContent:  "a\nb\nc\n",
			newContent:  "x\ny\nz\n",
			wantAdded:   3,
			wantRemoved: 3,
			wantNet:     0,
		},
		{
			name:        "net addition",
			oldContent:  "a\n",
			newContent:  "a\nb\nc\n",
			wantAdded:   2,
			wantRemoved: 0,
			wantNet:     2,
		},
		{
			name:        "net removal",
			oldContent:  "a\nb\nc\n",
			newContent:  "a\n",
			wantAdded:   0,
			wantRemoved: 2,
			wantNet:     -2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := GenerateStructuredDiff("test.txt", tt.oldContent, tt.newContent)

			if diff.Summary.LinesAdded != tt.wantAdded {
				t.Errorf("lines added = %d, want %d", diff.Summary.LinesAdded, tt.wantAdded)
			}

			if diff.Summary.LinesRemoved != tt.wantRemoved {
				t.Errorf("lines removed = %d, want %d", diff.Summary.LinesRemoved, tt.wantRemoved)
			}

			if diff.Summary.NetChange != tt.wantNet {
				t.Errorf("net change = %d, want %d", diff.Summary.NetChange, tt.wantNet)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "trailing newline",
			input: "a\nb\nc\n",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "no trailing newline",
			input: "a\nb\nc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "single line with newline",
			input: "hello\n",
			want:  []string{"hello"},
		},
		{
			name:  "single line without newline",
			input: "hello",
			want:  []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitLines(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
