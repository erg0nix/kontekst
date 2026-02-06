package agent

import (
	"errors"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

type mockContext struct {
	messages []core.Message
}

func (m *mockContext) SystemContent() string {
	return ""
}

func (m *mockContext) StartRun(systemContent string, systemTokens int) error {
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

func TestAddToolResults_SingleTool(t *testing.T) {
	ctx := &mockContext{}
	agent := &Agent{
		context:  ctx,
		provider: &mockProvider{},
	}

	batch := []toolExecution{
		{
			call:   &pendingCall{ID: "call1", Name: "test_tool"},
			output: "result",
			err:    nil,
		},
	}

	if err := agent.addToolResults(batch); err != nil {
		t.Fatalf("addToolResults failed: %v", err)
	}

	if len(ctx.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ctx.messages))
	}

	msg := ctx.messages[0]
	if msg.Role != core.RoleTool {
		t.Errorf("expected role 'tool', got %s", msg.Role)
	}

	if msg.Content != "result" {
		t.Errorf("expected content 'result', got %s", msg.Content)
	}

	if msg.ToolResult == nil {
		t.Fatal("expected ToolResult to be set")
	}

	if msg.ToolResult.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %s", msg.ToolResult.Name)
	}
}

func TestAddToolResults_BatchedTools(t *testing.T) {
	ctx := &mockContext{}
	agent := &Agent{
		context:  ctx,
		provider: &mockProvider{},
	}

	batch := []toolExecution{
		{
			call:   &pendingCall{ID: "call1", Name: "tool1"},
			output: "result1",
			err:    nil,
		},
		{
			call:   &pendingCall{ID: "call2", Name: "tool2"},
			output: "result2",
			err:    nil,
		},
		{
			call:   &pendingCall{ID: "call3", Name: "tool3"},
			output: "",
			err:    errors.New("tool error"),
		},
	}

	if err := agent.addToolResults(batch); err != nil {
		t.Fatalf("addToolResults failed: %v", err)
	}

	if len(ctx.messages) != 1 {
		t.Fatalf("expected 1 message (batched), got %d", len(ctx.messages))
	}

	msg := ctx.messages[0]
	if msg.Role != core.RoleTool {
		t.Errorf("expected role 'tool', got %s", msg.Role)
	}

	if msg.ToolResult == nil {
		t.Fatal("expected ToolResult to be set")
	}

	if msg.ToolResult.Name != "batch_tool_results" {
		t.Errorf("expected tool name 'batch_tool_results', got %s", msg.ToolResult.Name)
	}

	if msg.ToolResult.CallID != "call1,call2,call3" {
		t.Errorf("expected call_id 'call1,call2,call3', got %s", msg.ToolResult.CallID)
	}

	if !msg.ToolResult.IsError {
		t.Error("expected IsError to be true (batch contains error)")
	}

	content := msg.Content
	if content == "" {
		t.Error("expected non-empty content")
	}

	expectedSeparator := "\n\n---\n\n"
	separatorCount := 0
	for i := 0; i < len(content)-len(expectedSeparator); i++ {
		if content[i:i+len(expectedSeparator)] == expectedSeparator {
			separatorCount++
		}
	}

	if separatorCount != 2 {
		t.Errorf("expected 2 separators between 3 results, got %d", separatorCount)
	}
}
