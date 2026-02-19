package builtin

import (
	"context"
	"testing"

	toolpkg "github.com/erg0nix/kontekst/internal/tool"
)

func TestIsSafeRelative(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"file.txt", true},
		{"dir/file.txt", true},
		{"a/b/c/d.txt", true},
		{"", false},
		{"/absolute/path", false},
		{"\\windows\\path", false},
		{"../parent", false},
		{"dir/../escape", true},
		{"./current/dir", true},
		{"dir/./file", true},
		{"foo/../../bar", false},
	}

	for _, tt := range tests {
		got := isSafeRelative(tt.path)
		if got != tt.want {
			t.Errorf("isSafeRelative(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestGetStringArg(t *testing.T) {
	args := map[string]any{
		"str":   "hello",
		"int":   42,
		"empty": "",
		"bool":  true,
	}

	tests := []struct {
		key     string
		wantVal string
		wantOK  bool
	}{
		{"str", "hello", true},
		{"empty", "", true},
		{"int", "", false},
		{"bool", "", false},
		{"missing", "", false},
	}

	for _, tt := range tests {
		gotVal, gotOK := getStringArg(tt.key, args)
		if gotVal != tt.wantVal || gotOK != tt.wantOK {
			t.Errorf("getStringArg(%q) = (%q, %v), want (%q, %v)",
				tt.key, gotVal, gotOK, tt.wantVal, tt.wantOK)
		}
	}
}

func TestGetIntArg(t *testing.T) {
	args := map[string]any{
		"int":     42,
		"int64":   int64(100),
		"float64": float64(3.14),
		"str":     "42",
		"bool":    true,
	}

	tests := []struct {
		key     string
		wantVal int
		wantOK  bool
	}{
		{"int", 42, true},
		{"int64", 100, true},
		{"float64", 3, true},
		{"str", 0, false},
		{"bool", 0, false},
		{"missing", 0, false},
	}

	for _, tt := range tests {
		gotVal, gotOK := getIntArg(tt.key, args)
		if gotVal != tt.wantVal || gotOK != tt.wantOK {
			t.Errorf("getIntArg(%q) = (%d, %v), want (%d, %v)",
				tt.key, gotVal, gotOK, tt.wantVal, tt.wantOK)
		}
	}
}

func TestResolveBaseDir(t *testing.T) {
	fallback := "/fallback/dir"

	ctx := context.Background()
	if got := resolveBaseDir(ctx, fallback); got != fallback {
		t.Errorf("resolveBaseDir without working dir = %q, want %q", got, fallback)
	}

	ctxWithDir := toolpkg.WithWorkingDir(ctx, "/working/dir")
	if got := resolveBaseDir(ctxWithDir, fallback); got != "/working/dir" {
		t.Errorf("resolveBaseDir with working dir = %q, want /working/dir", got)
	}
}
