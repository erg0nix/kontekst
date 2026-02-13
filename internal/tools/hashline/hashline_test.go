package hashline

import (
	"strings"
	"testing"
)

func TestGenerateHashMap(t *testing.T) {
	t.Run("no collisions", func(t *testing.T) {
		lines := []string{"line 1", "line 2", "line 3"}
		hashMap, warning := GenerateHashMap(lines)

		if len(hashMap) != 3 {
			t.Errorf("hashMap size = %d, want 3", len(hashMap))
		}

		if warning != "" {
			t.Errorf("expected no warning, got: %q", warning)
		}

		seenHashes := make(map[string]bool)
		for _, hash := range hashMap {
			if seenHashes[hash] {
				t.Errorf("duplicate hash: %q", hash)
			}
			seenHashes[hash] = true
		}
	})

	t.Run("with collisions - disambiguation", func(t *testing.T) {
		lines := []string{"dup", "dup", "dup"}
		hashMap, warning := GenerateHashMap(lines)

		if warning == "" {
			t.Error("expected collision warning for duplicate lines")
		}

		hash1 := hashMap[1]
		hash2 := hashMap[2]
		hash3 := hashMap[3]

		if strings.Contains(hash1, ".") {
			t.Errorf("first occurrence should not have suffix, got: %q", hash1)
		}

		if !strings.Contains(hash2, ".") || !strings.Contains(hash3, ".") {
			t.Errorf("subsequent occurrences should have suffix, got: %q, %q", hash2, hash3)
		}

		if hash1 == hash2 || hash1 == hash3 || hash2 == hash3 {
			t.Errorf("hashes not unique after disambiguation: %q, %q, %q", hash1, hash2, hash3)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		lines := []string{}
		hashMap, warning := GenerateHashMap(lines)

		if len(hashMap) != 0 {
			t.Errorf("hashMap size = %d, want 0", len(hashMap))
		}

		if warning != "" {
			t.Errorf("expected no warning, got: %q", warning)
		}
	})
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
				_ = ComputeLineHash(line)
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
				_, _ = GenerateHashMap(lines)
			}
		})
	}
}
