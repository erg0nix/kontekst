package sessions

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestContextLogWriter_Write(t *testing.T) {
	dir := t.TempDir()
	writer := NewContextLogWriter(dir)

	runID := core.RunID("test-run-1")
	snapshot := core.ContextSnapshot{
		ContextSize:  4096,
		SystemTokens: 100,
		TotalTokens:  200,
	}

	if err := writer.Write(runID, 1, snapshot); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	path := filepath.Join(dir, "runs", string(runID), "context.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	var entry ContextLogEntry
	if err := json.NewDecoder(file).Decode(&entry); err != nil {
		t.Fatalf("failed to decode entry: %v", err)
	}

	if entry.RunID != runID {
		t.Errorf("RunID: got %s, want %s", entry.RunID, runID)
	}
	if entry.Turn != 1 {
		t.Errorf("Turn: got %d, want 1", entry.Turn)
	}
	if entry.Snapshot.ContextSize != 4096 {
		t.Errorf("Snapshot.ContextSize: got %d, want 4096", entry.Snapshot.ContextSize)
	}
	if entry.Snapshot.SystemTokens != 100 {
		t.Errorf("Snapshot.SystemTokens: got %d, want 100", entry.Snapshot.SystemTokens)
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestContextLogWriter_MultipleWrites(t *testing.T) {
	dir := t.TempDir()
	writer := NewContextLogWriter(dir)

	runID := core.RunID("test-run-2")

	for i := 1; i <= 3; i++ {
		snapshot := core.ContextSnapshot{
			ContextSize: 4096,
			TotalTokens: i * 100,
		}
		if err := writer.Write(runID, i, snapshot); err != nil {
			t.Fatalf("Write turn %d failed: %v", i, err)
		}
	}

	path := filepath.Join(dir, "runs", string(runID), "context.jsonl")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
		var entry ContextLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("failed to decode line %d: %v", count, err)
		}
		if entry.Turn != count {
			t.Errorf("line %d: Turn got %d, want %d", count, entry.Turn, count)
		}
		if entry.Snapshot.TotalTokens != count*100 {
			t.Errorf("line %d: TotalTokens got %d, want %d", count, entry.Snapshot.TotalTokens, count*100)
		}
	}

	if count != 3 {
		t.Errorf("expected 3 JSONL lines, got %d", count)
	}
}
