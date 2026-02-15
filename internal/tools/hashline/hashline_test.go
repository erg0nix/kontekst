package hashline

import (
	"fmt"
	"testing"
)

func TestGenerateHashMap(t *testing.T) {
	t.Run("unique lines", func(t *testing.T) {
		lines := []string{"line 1", "line 2", "line 3"}
		hashMap := GenerateHashMap(lines)

		if len(hashMap) != 3 {
			t.Errorf("hashMap size = %d, want 3", len(hashMap))
		}

		for lineNum, hash := range hashMap {
			if hash == "" {
				t.Errorf("line %d has empty hash", lineNum)
			}
			if len(hash) != 3 {
				t.Errorf("line %d hash length = %d, want 3", lineNum, len(hash))
			}
		}
	})

	t.Run("duplicate lines get same hash", func(t *testing.T) {
		lines := []string{"dup", "dup", "dup"}
		hashMap := GenerateHashMap(lines)

		if hashMap[1] != hashMap[2] || hashMap[2] != hashMap[3] {
			t.Errorf("duplicate lines should produce same hash: %q, %q, %q",
				hashMap[1], hashMap[2], hashMap[3])
		}
	})

	t.Run("empty file", func(t *testing.T) {
		lines := []string{}
		hashMap := GenerateHashMap(lines)

		if len(hashMap) != 0 {
			t.Errorf("hashMap size = %d, want 0", len(hashMap))
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
		b.Run(fmt.Sprintf("len=%d", len(line)), func(b *testing.B) {
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

		b.Run(fmt.Sprintf("lines=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = GenerateHashMap(lines)
			}
		})
	}
}
