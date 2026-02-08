package agent

import (
	"context"
	"testing"

	ctx "github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
)

type mockContext struct {
	messages []core.Message
}

func (m *mockContext) SystemContent() string {
	return ""
}

func (m *mockContext) StartRun(params ctx.BudgetParams) error {
	return nil
}

func (m *mockContext) CompleteRun() error {
	return nil
}

func (m *mockContext) BuildContext() ([]core.Message, error) {
	return m.messages, nil
}

func (m *mockContext) AddMessage(msg core.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockContext) SetActiveSkill(skill *core.SkillMetadata) {}

func (m *mockContext) ActiveSkill() *core.SkillMetadata {
	return nil
}

func (m *mockContext) RenderUserMessage(prompt string) (string, error) {
	return prompt, nil
}

func (m *mockContext) SetAgentSystemPrompt(prompt string) {}

func (m *mockContext) Snapshot() core.ContextSnapshot {
	return core.ContextSnapshot{}
}

type mockToolExecutor struct{}

func (m *mockToolExecutor) Execute(name string, args map[string]any, ctx context.Context) (string, error) {
	return "mock output", nil
}

func (m *mockToolExecutor) ToolDefinitions() []core.ToolDef {
	return nil
}

func (m *mockToolExecutor) Preview(name string, args map[string]any, ctx context.Context) (string, error) {
	return "", nil
}

type mockProvider struct{}

func (m *mockProvider) GenerateChat(messages []core.Message, tools []core.ToolDef, sampling *core.SamplingConfig, model string, useToolRole bool) (core.ChatResponse, error) {
	return core.ChatResponse{}, nil
}

func (m *mockProvider) CountTokens(text string) (int, error) {
	return len(text) / 4, nil
}

func (m *mockProvider) ConcurrencyLimit() int {
	return 1
}

func TestExecuteTools_SingleTool(t *testing.T) {
	ctx := &mockContext{}
	eventCh := make(chan AgentEvent, 32)
	ag := &Agent{
		context:  ctx,
		provider: &mockProvider{},
		tools:    &mockToolExecutor{},
	}

	approved := true
	calls := []*pendingCall{
		{ID: "call1", Name: "test_tool", Args: map[string]any{}, Approved: &approved},
	}

	if err := ag.executeTools("run1", "batch1", calls, eventCh); err != nil {
		t.Fatalf("executeTools failed: %v", err)
	}

	if len(ctx.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ctx.messages))
	}

	msg := ctx.messages[0]
	if msg.Role != core.RoleTool {
		t.Errorf("expected role 'tool', got %s", msg.Role)
	}
	if msg.ToolResult == nil {
		t.Fatal("expected ToolResult to be set")
	}
	if msg.ToolResult.CallID != "call1" {
		t.Errorf("expected call_id 'call1', got %s", msg.ToolResult.CallID)
	}
}

func TestExecuteTools_MultipleToolsProduceIndividualMessages(t *testing.T) {
	ctx := &mockContext{}
	eventCh := make(chan AgentEvent, 32)
	ag := &Agent{
		context:  ctx,
		provider: &mockProvider{},
		tools:    &mockToolExecutor{},
	}

	approved := true
	calls := []*pendingCall{
		{ID: "call1", Name: "tool1", Args: map[string]any{}, Approved: &approved},
		{ID: "call2", Name: "tool2", Args: map[string]any{}, Approved: &approved},
	}

	if err := ag.executeTools("run1", "batch1", calls, eventCh); err != nil {
		t.Fatalf("executeTools failed: %v", err)
	}

	if len(ctx.messages) != 2 {
		t.Fatalf("expected 2 messages (one per tool), got %d", len(ctx.messages))
	}

	for i, msg := range ctx.messages {
		if msg.Role != core.RoleTool {
			t.Errorf("message %d: expected role 'tool', got %s", i, msg.Role)
		}
		if msg.ToolResult == nil {
			t.Errorf("message %d: expected ToolResult to be set", i)
		}
	}

	if ctx.messages[0].ToolResult.CallID != "call1" {
		t.Errorf("expected first message call_id 'call1', got %s", ctx.messages[0].ToolResult.CallID)
	}
	if ctx.messages[1].ToolResult.CallID != "call2" {
		t.Errorf("expected second message call_id 'call2', got %s", ctx.messages[1].ToolResult.CallID)
	}
}

func TestExecuteTools_DeniedToolRecordsError(t *testing.T) {
	ctx := &mockContext{}
	eventCh := make(chan AgentEvent, 32)
	ag := &Agent{
		context:  ctx,
		provider: &mockProvider{},
	}

	denied := false
	calls := []*pendingCall{
		{ID: "call1", Name: "tool1", Args: map[string]any{}, Approved: &denied, Reason: "not allowed"},
	}

	if err := ag.executeTools("run1", "batch1", calls, eventCh); err != nil {
		t.Fatalf("executeTools failed: %v", err)
	}

	if len(ctx.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ctx.messages))
	}

	msg := ctx.messages[0]
	if !msg.ToolResult.IsError {
		t.Error("expected IsError to be true for denied tool")
	}
}

func TestExecuteTools_MixedApprovedAndDenied(t *testing.T) {
	ctx := &mockContext{}
	eventCh := make(chan AgentEvent, 32)
	ag := &Agent{
		context:  ctx,
		provider: &mockProvider{},
		tools:    &mockToolExecutor{},
	}

	approved := true
	denied := false
	calls := []*pendingCall{
		{ID: "call1", Name: "tool1", Args: map[string]any{}, Approved: &approved},
		{ID: "call2", Name: "tool2", Args: map[string]any{}, Approved: &denied, Reason: "denied"},
		{ID: "call3", Name: "tool3", Args: map[string]any{}, Approved: &approved},
	}

	if err := ag.executeTools("run1", "batch1", calls, eventCh); err != nil {
		t.Fatalf("executeTools failed: %v", err)
	}

	if len(ctx.messages) != 3 {
		t.Fatalf("expected 3 messages (one per tool), got %d", len(ctx.messages))
	}

	if ctx.messages[1].ToolResult.IsError != true {
		t.Error("expected second message to be an error (denied)")
	}

	if ctx.messages[0].ToolResult.IsError != false {
		t.Error("expected first message to not be an error")
	}
}
