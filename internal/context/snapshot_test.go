package context

import (
	"path/filepath"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestContextWindow_Snapshot(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	historicalMessages := []core.Message{
		{Role: core.RoleUser, Content: "past user msg", Tokens: 10},
		{Role: core.RoleAssistant, Content: "past assistant msg", Tokens: 20},
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	systemTokens := 100
	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: systemTokens}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleUser, Content: "memory msg", Tokens: 15}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	snapshot := cw.Snapshot()

	if snapshot.ContextSize != 4096 {
		t.Errorf("ContextSize: got %d, want 4096", snapshot.ContextSize)
	}
	if snapshot.SystemTokens != systemTokens {
		t.Errorf("SystemTokens: got %d, want %d", snapshot.SystemTokens, systemTokens)
	}
	if snapshot.HistoryTokens != 30 {
		t.Errorf("HistoryTokens: got %d, want 30", snapshot.HistoryTokens)
	}
	if snapshot.MemoryTokens != 15 {
		t.Errorf("MemoryTokens: got %d, want 15", snapshot.MemoryTokens)
	}
	if snapshot.TotalTokens != 145 {
		t.Errorf("TotalTokens: got %d, want 145", snapshot.TotalTokens)
	}
	if snapshot.RemainingTokens != 4096-145 {
		t.Errorf("RemainingTokens: got %d, want %d", snapshot.RemainingTokens, 4096-145)
	}
	if snapshot.HistoryMessages != 2 {
		t.Errorf("HistoryMessages: got %d, want 2", snapshot.HistoryMessages)
	}
	if snapshot.MemoryMessages != 1 {
		t.Errorf("MemoryMessages: got %d, want 1", snapshot.MemoryMessages)
	}
	if snapshot.TotalMessages != 4 {
		t.Errorf("TotalMessages: got %d, want 4", snapshot.TotalMessages)
	}

	expectedBudget := 4096 - systemTokens - 15
	if snapshot.HistoryBudget != expectedBudget {
		t.Errorf("HistoryBudget: got %d, want %d", snapshot.HistoryBudget, expectedBudget)
	}
}

func TestContextWindow_SnapshotEmpty(t *testing.T) {
	cw := newTestContextWindow(t)

	systemTokens := 50
	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: systemTokens}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	snapshot := cw.Snapshot()

	if snapshot.SystemTokens != systemTokens {
		t.Errorf("SystemTokens: got %d, want %d", snapshot.SystemTokens, systemTokens)
	}
	if snapshot.HistoryTokens != 0 {
		t.Errorf("HistoryTokens: got %d, want 0", snapshot.HistoryTokens)
	}
	if snapshot.MemoryTokens != 0 {
		t.Errorf("MemoryTokens: got %d, want 0", snapshot.MemoryTokens)
	}
	if snapshot.TotalTokens != systemTokens {
		t.Errorf("TotalTokens: got %d, want %d", snapshot.TotalTokens, systemTokens)
	}
	if snapshot.HistoryMessages != 0 {
		t.Errorf("HistoryMessages: got %d, want 0", snapshot.HistoryMessages)
	}
	if snapshot.MemoryMessages != 0 {
		t.Errorf("MemoryMessages: got %d, want 0", snapshot.MemoryMessages)
	}
	if snapshot.TotalMessages != 1 {
		t.Errorf("TotalMessages: got %d, want 1 (system only)", snapshot.TotalMessages)
	}
}

func TestContextWindow_SnapshotMessageDetails(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	historicalMessages := []core.Message{
		{Role: core.RoleUser, Content: "history", Tokens: 10},
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 50}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleAssistant, Content: "memory", Tokens: 25}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	snapshot := cw.Snapshot()

	if len(snapshot.Messages) != 3 {
		t.Fatalf("expected 3 message stats, got %d", len(snapshot.Messages))
	}

	if snapshot.Messages[0].Role != core.RoleSystem || snapshot.Messages[0].Source != "system" || snapshot.Messages[0].Tokens != 50 {
		t.Errorf("message 0: got %+v, want system/system/50", snapshot.Messages[0])
	}
	if snapshot.Messages[1].Role != core.RoleUser || snapshot.Messages[1].Source != "history" || snapshot.Messages[1].Tokens != 10 {
		t.Errorf("message 1: got %+v, want user/history/10", snapshot.Messages[1])
	}
	if snapshot.Messages[2].Role != core.RoleAssistant || snapshot.Messages[2].Source != "memory" || snapshot.Messages[2].Tokens != 25 {
		t.Errorf("message 2: got %+v, want assistant/memory/25", snapshot.Messages[2])
	}
}
