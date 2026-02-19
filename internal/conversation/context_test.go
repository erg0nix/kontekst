package conversation

import (
	"path/filepath"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestContextWindow_SystemPromptFirst(t *testing.T) {
	cw := newTestContextWindow(t)
	systemContent := "You are a helpful assistant."

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: systemContent, SystemTokens: 10}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleUser, Content: "hello", Tokens: 5}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) < 1 {
		t.Fatal("expected at least one message")
	}

	if msgs[0].Role != core.RoleSystem {
		t.Errorf("first message should be system, got %v", msgs[0].Role)
	}

	if msgs[0].Content != systemContent {
		t.Errorf("system content mismatch: got %q, want %q", msgs[0].Content, systemContent)
	}
}

func TestContextWindow_SystemContentWithActiveSkill(t *testing.T) {
	cw := newTestContextWindow(t)
	cw.SetAgentSystemPrompt("Base system prompt.")
	cw.SetActiveSkill(&core.SkillMetadata{Name: "test-skill", Path: "/path/to/skill"})

	content := cw.SystemContent()

	if content == "Base system prompt." {
		t.Error("expected active skill to be appended to system content")
	}

	expected := "Base system prompt.\n\n<active-skill name=\"test-skill\" path=\"/path/to/skill\" />"
	if content != expected {
		t.Errorf("system content mismatch:\ngot:  %q\nwant: %q", content, expected)
	}
}

func TestContextWindow_HistoryFromFile(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	historicalMessages := []core.Message{
		{Role: core.RoleUser, Content: "past user msg", Tokens: 10},
		{Role: core.RoleAssistant, Content: "past assistant msg", Tokens: 20},
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 100}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system + 2 history), got %d", len(msgs))
	}

	if msgs[1].Content != "past user msg" {
		t.Errorf("expected history message, got %q", msgs[1].Content)
	}
}

func TestContextWindow_AddMessageGoesToMemoryAndFile(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 10}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msg := core.Message{Role: core.RoleUser, Content: "new message", Tokens: 15}
	if err := cw.AddMessage(msg); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system + memory), got %d", len(msgs))
	}

	if msgs[1].Content != "new message" {
		t.Errorf("expected memory message, got %q", msgs[1].Content)
	}

	fileMsgs, err := sf.LoadTail(100000)
	if err != nil {
		t.Fatalf("LoadTail failed: %v", err)
	}

	if len(fileMsgs) != 1 || fileMsgs[0].Content != "new message" {
		t.Errorf("message not persisted to file correctly: %v", fileMsgs)
	}
}

func TestContextWindow_BuildContextOrder(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	historicalMessages := []core.Message{
		{Role: core.RoleUser, Content: "history1", Tokens: 10},
		{Role: core.RoleAssistant, Content: "history2", Tokens: 10},
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system content", SystemTokens: 50}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleUser, Content: "memory1", Tokens: 10}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleAssistant, Content: "memory2", Tokens: 10}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	expected := []string{"system content", "history1", "history2", "memory1", "memory2"}
	if len(msgs) != len(expected) {
		t.Fatalf("expected %d messages, got %d", len(expected), len(msgs))
	}

	for i, exp := range expected {
		if msgs[i].Content != exp {
			t.Errorf("message %d: expected %q, got %q", i, exp, msgs[i].Content)
		}
	}
}

func TestContextWindow_CompleteRunClearsMemory(t *testing.T) {
	cw := newTestContextWindow(t)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 10}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if err := cw.AddMessage(core.Message{Role: core.RoleUser, Content: "memory msg", Tokens: 10}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	cw.CompleteRun()

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("expected only system message after CompleteRun, got %d messages", len(msgs))
	}
}

func TestContextWindow_MultipleRunsAccumulateHistory(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 50}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}
	if err := cw.AddMessage(core.Message{Role: core.RoleUser, Content: "run1", Tokens: 10}); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}
	cw.CompleteRun()

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 50}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system + history from run1), got %d", len(msgs))
	}

	if msgs[1].Content != "run1" {
		t.Errorf("expected history from run1, got %q", msgs[1].Content)
	}
}

func TestContextWindow_HistoryBudgetCalculation(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	var historicalMessages []core.Message
	for i := 0; i < 100; i++ {
		historicalMessages = append(historicalMessages, core.Message{
			Role:    core.RoleUser,
			Content: "history",
			Tokens:  100,
		})
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	systemTokens := 100
	if err := cw.StartRun(BudgetParams{ContextSize: 1000, SystemContent: "system", SystemTokens: systemTokens}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	historyCount := len(msgs) - 1
	historyTokens := historyCount * 100

	maxHistoryBudget := 1000 - systemTokens

	if historyTokens > maxHistoryBudget+100 {
		t.Errorf("history tokens (%d) exceeds budget (%d)", historyTokens, maxHistoryBudget)
	}
}

func TestContextWindow_LargeSystemPromptLeavesRoomForHistory(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	historicalMessages := []core.Message{
		{Role: core.RoleUser, Content: "history", Tokens: 50},
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	largeSystemTokens := 800
	if err := cw.StartRun(BudgetParams{ContextSize: 1000, SystemContent: "large system prompt", SystemTokens: largeSystemTokens}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	maxHistoryBudget := 1000 - largeSystemTokens

	if maxHistoryBudget < 50 {
		if len(msgs) != 1 {
			t.Errorf("expected only system message when budget too small, got %d", len(msgs))
		}
	} else if len(msgs) != 2 {
		t.Errorf("expected system + history message, got %d", len(msgs))
	}
}

func TestContextWindow_EmptySession(t *testing.T) {
	cw := newTestContextWindow(t)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 10}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("expected only system message for empty session, got %d", len(msgs))
	}

	if msgs[0].Role != core.RoleSystem {
		t.Errorf("expected system message, got %v", msgs[0].Role)
	}
}

func TestContextWindow_SessionWithOnlyToolMessages(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	toolMessages := []core.Message{
		{Role: core.RoleTool, Content: "tool result 1", Tokens: 10},
		{Role: core.RoleTool, Content: "tool result 2", Tokens: 10},
	}
	writeMessages(t, sessionPath, toolMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 4096, SystemContent: "system", SystemTokens: 50}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system + 2 tool), got %d", len(msgs))
	}

	for i := 1; i < len(msgs); i++ {
		if msgs[i].Role != core.RoleTool {
			t.Errorf("message %d: expected tool role, got %v", i, msgs[i].Role)
		}
	}
}

func TestContextWindow_VeryLargeContextSize(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")

	var historicalMessages []core.Message
	for i := 0; i < 10; i++ {
		historicalMessages = append(historicalMessages, core.Message{
			Role:    core.RoleUser,
			Content: "history",
			Tokens:  100,
		})
	}
	writeMessages(t, sessionPath, historicalMessages...)

	sf := NewSessionFile(sessionPath)
	cw := newContextWindow(sf)

	if err := cw.StartRun(BudgetParams{ContextSize: 1000000, SystemContent: "system", SystemTokens: 100}); err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	msgs, err := cw.BuildContext()
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}

	if len(msgs) != 11 {
		t.Errorf("expected all 10 history + system, got %d", len(msgs))
	}
}

func TestContextWindow_ActiveSkill(t *testing.T) {
	cw := newTestContextWindow(t)

	if skill := cw.ActiveSkill(); skill != nil {
		t.Error("expected nil active skill initially")
	}

	expectedSkill := &core.SkillMetadata{Name: "test", Path: "/test"}
	cw.SetActiveSkill(expectedSkill)

	if skill := cw.ActiveSkill(); skill == nil || skill.Name != "test" {
		t.Errorf("expected skill 'test', got %v", skill)
	}

	cw.SetActiveSkill(nil)
	if skill := cw.ActiveSkill(); skill != nil {
		t.Error("expected nil after clearing active skill")
	}
}

func newTestContextWindow(t *testing.T) *contextWindow {
	t.Helper()

	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	sf := NewSessionFile(sessionPath)

	return newContextWindow(sf)
}
