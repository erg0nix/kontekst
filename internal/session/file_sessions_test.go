package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

func newTestService(t *testing.T) *FileSessionService {
	t.Helper()
	return &FileSessionService{BaseDir: t.TempDir()}
}

func createSessionFile(t *testing.T, svc *FileSessionService, id core.SessionID, content string) {
	t.Helper()
	dir := svc.sessionDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := svc.sessionPath(id)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestList_Empty(t *testing.T) {
	svc := newTestService(t)

	list, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list != nil {
		t.Fatalf("expected nil, got %v", list)
	}

	if err := os.MkdirAll(svc.sessionDir(), 0o755); err != nil {
		t.Fatal(err)
	}

	list, err = svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list != nil {
		t.Fatalf("expected nil for empty dir, got %v", list)
	}
}

func TestList_MultipleSessions(t *testing.T) {
	svc := newTestService(t)

	createSessionFile(t, svc, "sess_20250101T000000.000000000_aaaaaaaaaaaa", "line1\nline2\n")
	createSessionFile(t, svc, "sess_20250102T000000.000000000_bbbbbbbbbbbb", "line1\n")

	if err := svc.SetDefaultAgent("sess_20250101T000000.000000000_aaaaaaaaaaaa", "coder"); err != nil {
		t.Fatal(err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}

	ids := map[core.SessionID]bool{}
	for _, info := range list {
		ids[info.ID] = true
	}

	if !ids["sess_20250101T000000.000000000_aaaaaaaaaaaa"] {
		t.Fatal("missing session aaa")
	}
	if !ids["sess_20250102T000000.000000000_bbbbbbbbbbbb"] {
		t.Fatal("missing session bbb")
	}
}

func TestList_SortedByModified(t *testing.T) {
	svc := newTestService(t)

	createSessionFile(t, svc, "sess_20250101T000000.000000000_aaaaaaaaaaaa", "old\n")

	oldPath := svc.sessionPath("sess_20250101T000000.000000000_aaaaaaaaaaaa")
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	createSessionFile(t, svc, "sess_20250102T000000.000000000_bbbbbbbbbbbb", "new\n")

	list, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}

	if list[0].ID != "sess_20250102T000000.000000000_bbbbbbbbbbbb" {
		t.Fatalf("expected newest first, got %s", list[0].ID)
	}
}

func TestGet_ExistingSession(t *testing.T) {
	svc := newTestService(t)
	id := core.SessionID("sess_20250212T123045.000000000_a1b2c3d4e5f6")

	createSessionFile(t, svc, id, "msg1\nmsg2\nmsg3\n")
	if err := svc.SetDefaultAgent(id, "coder"); err != nil {
		t.Fatal(err)
	}

	info, err := svc.Get(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ID != id {
		t.Fatalf("expected ID %s, got %s", id, info.ID)
	}
	if info.MessageCount != 3 {
		t.Fatalf("expected 3 messages, got %d", info.MessageCount)
	}
	if info.DefaultAgent != "coder" {
		t.Fatalf("expected agent 'coder', got %q", info.DefaultAgent)
	}
	if info.FileSize == 0 {
		t.Fatal("expected non-zero file size")
	}
	if info.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
	if info.CreatedAt.Year() != 2025 || info.CreatedAt.Month() != 2 || info.CreatedAt.Day() != 12 {
		t.Fatalf("unexpected CreatedAt: %v", info.CreatedAt)
	}
	if info.ModifiedAt.IsZero() {
		t.Fatal("expected non-zero ModifiedAt")
	}
}

func TestGet_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get("sess_nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "session not found") {
		t.Fatalf("expected 'session not found', got %q", err.Error())
	}
}

func TestGet_EmptySession(t *testing.T) {
	svc := newTestService(t)
	id := core.SessionID("sess_20250101T000000.000000000_aaaaaaaaaaaa")

	createSessionFile(t, svc, id, "")

	info, err := svc.Get(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.MessageCount != 0 {
		t.Fatalf("expected 0 messages, got %d", info.MessageCount)
	}
	if info.FileSize != 0 {
		t.Fatalf("expected 0 size, got %d", info.FileSize)
	}
}

func TestDelete_RemovesBothFiles(t *testing.T) {
	svc := newTestService(t)
	id := core.SessionID("sess_20250101T000000.000000000_aaaaaaaaaaaa")

	createSessionFile(t, svc, id, "data\n")
	if err := svc.SetDefaultAgent(id, "test"); err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(svc.sessionPath(id)); !os.IsNotExist(err) {
		t.Fatal("expected .jsonl to be removed")
	}
	if _, err := os.Stat(svc.metaPath(id)); !os.IsNotExist(err) {
		t.Fatal("expected .meta.json to be removed")
	}
}

func TestDelete_NoMeta(t *testing.T) {
	svc := newTestService(t)
	id := core.SessionID("sess_20250101T000000.000000000_aaaaaaaaaaaa")

	createSessionFile(t, svc, id, "data\n")

	if err := svc.Delete(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(svc.sessionPath(id)); !os.IsNotExist(err) {
		t.Fatal("expected .jsonl to be removed")
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete("sess_nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "session not found") {
		t.Fatalf("expected 'session not found', got %q", err.Error())
	}
}

func TestParseSessionTimestamp(t *testing.T) {
	tests := []struct {
		id   core.SessionID
		year int
		zero bool
	}{
		{"sess_20250212T123045.000000000_a1b2c3d4e5f6", 2025, false},
		{"sess_20240101T000000.000000000_ffffffffffff", 2024, false},
		{"malformed", 0, true},
		{"sess_badtimestamp_abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		result := parseSessionTimestamp(tt.id)
		if tt.zero && !result.IsZero() {
			t.Errorf("parseSessionTimestamp(%q) = %v, want zero", tt.id, result)
		}
		if !tt.zero && result.Year() != tt.year {
			t.Errorf("parseSessionTimestamp(%q).Year() = %d, want %d", tt.id, result.Year(), tt.year)
		}
	}
}

func TestList_IgnoresMetaFiles(t *testing.T) {
	svc := newTestService(t)
	id := core.SessionID("sess_20250101T000000.000000000_aaaaaaaaaaaa")

	createSessionFile(t, svc, id, "data\n")
	if err := svc.SetDefaultAgent(id, "test"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(svc.sessionDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files (.jsonl + .meta.json), got %d", len(entries))
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 session (ignoring .meta.json), got %d", len(list))
	}
}

func TestCountLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := countLines(path); n != 3 {
		t.Fatalf("expected 3 lines, got %d", n)
	}

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := countLines(path); n != 0 {
		t.Fatalf("expected 0 lines, got %d", n)
	}
}
