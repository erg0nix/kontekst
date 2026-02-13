package builtin

import (
	"strings"
	"testing"
)

func TestComputeLineHash(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "empty line",
			line: "",
		},
		{
			name: "simple line",
			line: "hello world",
		},
		{
			name: "line with special chars",
			line: "func main() {",
		},
		{
			name: "line with whitespace",
			line: "    indented line",
		},
		{
			name: "line with unicode",
			line: "こんにちは世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := computeLineHash(tt.line)
			hash2 := computeLineHash(tt.line)

			if hash1 != hash2 {
				t.Errorf("hash not deterministic: got %q and %q for same input", hash1, hash2)
			}

			if len(hash1) != 3 {
				t.Errorf("hash length = %d, want 3", len(hash1))
			}

			for _, c := range hash1 {
				if !isBase64URLSafe(c) {
					t.Errorf("hash contains non-URL-safe character: %q", c)
				}
			}
		})
	}
}

func TestComputeLineHashUniqueness(t *testing.T) {
	lines := []string{
		"line 1",
		"line 2",
		"line 3",
		"different content",
		"    indented",
	}

	hashes := make(map[string]bool)
	for _, line := range lines {
		hash := computeLineHash(line)
		if hashes[hash] {
			t.Logf("collision detected for line %q (hash: %s) - this is expected to be rare", line, hash)
		}
		hashes[hash] = true
	}
}

func TestDetectCollisions(t *testing.T) {
	tests := []struct {
		name            string
		lines           []string
		wantCollisions  bool
		minCollisionLen int
	}{
		{
			name:           "no collisions - unique lines",
			lines:          []string{"line 1", "line 2", "line 3"},
			wantCollisions: false,
		},
		{
			name:            "collision - duplicate lines",
			lines:           []string{"same", "same", "different"},
			wantCollisions:  true,
			minCollisionLen: 2,
		},
		{
			name:            "collision - multiple duplicates",
			lines:           []string{"a", "a", "a", "b"},
			wantCollisions:  true,
			minCollisionLen: 3,
		},
		{
			name:           "empty lines",
			lines:          []string{},
			wantCollisions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collisions := detectCollisions(tt.lines)

			if tt.wantCollisions {
				if len(collisions) == 0 {
					t.Errorf("expected collisions, got none")
				}
				for hash, indices := range collisions {
					if len(indices) < tt.minCollisionLen {
						t.Errorf("hash %q has %d lines, want at least %d", hash, len(indices), tt.minCollisionLen)
					}
				}
			} else {
				if len(collisions) > 0 {
					t.Errorf("expected no collisions, got %d", len(collisions))
				}
			}
		})
	}
}

func TestDisambiguateHash(t *testing.T) {
	tests := []struct {
		name       string
		hash       string
		occurrence int
		want       string
	}{
		{
			name:       "first occurrence",
			hash:       "a3b",
			occurrence: 0,
			want:       "a3b",
		},
		{
			name:       "second occurrence",
			hash:       "a3b",
			occurrence: 1,
			want:       "a3b.1",
		},
		{
			name:       "third occurrence",
			hash:       "a3b",
			occurrence: 2,
			want:       "a3b.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := disambiguateHash(tt.hash, tt.occurrence)
			if got != tt.want {
				t.Errorf("disambiguateHash(%q, %d) = %q, want %q", tt.hash, tt.occurrence, got, tt.want)
			}
		})
	}
}

func TestGenerateHashMap(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		wantWarning bool
	}{
		{
			name:        "no collisions",
			lines:       []string{"line 1", "line 2", "line 3"},
			wantWarning: false,
		},
		{
			name:        "with collisions",
			lines:       []string{"same", "same", "different"},
			wantWarning: true,
		},
		{
			name:        "empty file",
			lines:       []string{},
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashMap, warning := generateHashMap(tt.lines)

			if len(hashMap) != len(tt.lines) {
				t.Errorf("hashMap size = %d, want %d", len(hashMap), len(tt.lines))
			}

			for lineNum := range hashMap {
				if lineNum < 1 || lineNum > len(tt.lines) {
					t.Errorf("invalid line number: %d (file has %d lines)", lineNum, len(tt.lines))
				}
			}

			if tt.wantWarning && warning == "" {
				t.Errorf("expected collision warning, got none")
			}
			if !tt.wantWarning && warning != "" {
				t.Errorf("expected no warning, got: %q", warning)
			}

			seenHashes := make(map[string]bool)
			for _, hash := range hashMap {
				if seenHashes[hash] {
					t.Errorf("duplicate hash after disambiguation: %q", hash)
				}
				seenHashes[hash] = true
			}
		})
	}
}

func TestGenerateHashMapDisambiguation(t *testing.T) {
	lines := []string{"dup", "dup", "dup"}
	hashMap, warning := generateHashMap(lines)

	if warning == "" {
		t.Error("expected collision warning for duplicate lines")
	}

	hash1 := hashMap[1]
	hash2 := hashMap[2]
	hash3 := hashMap[3]

	if strings.Contains(hash1, ".") {
		t.Errorf("first occurrence should not have suffix, got: %q", hash1)
	}

	if !strings.Contains(hash2, ".") {
		t.Errorf("second occurrence should have suffix, got: %q", hash2)
	}
	if !strings.Contains(hash3, ".") {
		t.Errorf("third occurrence should have suffix, got: %q", hash3)
	}

	if hash1 == hash2 || hash1 == hash3 || hash2 == hash3 {
		t.Errorf("hashes not unique after disambiguation: %q, %q, %q", hash1, hash2, hash3)
	}
}

func BenchmarkComputeLineHash(b *testing.B) {
	lines := []string{
		"short",
		"medium length line with some content",
		"very long line with lots of content that goes on and on and on and on and on",
		"func (tool *EditFile) Execute(args map[string]any, ctx context.Context) (string, error) {",
	}

	for _, line := range lines {
		b.Run("len="+string(rune('0'+len(line)/10)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = computeLineHash(line)
			}
		})
	}
}

func BenchmarkGenerateHashMap(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		lines := make([]string, size)
		for i := range lines {
			lines[i] = "line content " + string(rune('0'+i%10))
		}

		b.Run("lines="+string(rune('0'+size/100)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = generateHashMap(lines)
			}
		})
	}
}

func isBase64URLSafe(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
