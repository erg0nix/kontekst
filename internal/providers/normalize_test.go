package providers

import (
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestNormalizeMessages(t *testing.T) {
	tests := []struct {
		name        string
		messages    []core.Message
		useToolRole bool
		expected    int
		checkRoles  bool
	}{
		{
			name: "consecutive user messages merged",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "first"},
				{Role: core.RoleUser, Content: "second"},
				{Role: core.RoleAssistant, Content: "response"},
			},
			useToolRole: false,
			expected:    3,
			checkRoles:  true,
		},
		{
			name: "consecutive tool messages kept separate (useToolRole=true)",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "prompt"},
				{Role: core.RoleAssistant, Content: "calling tools"},
				{Role: core.RoleTool, Content: "result1"},
				{Role: core.RoleTool, Content: "result2"},
				{Role: core.RoleAssistant, Content: "response"},
			},
			useToolRole: true,
			expected:    6,
		},
		{
			name: "consecutive tool results become user messages (useToolRole=false)",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "prompt"},
				{Role: core.RoleAssistant, Content: "calling tools"},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{Name: "tool1", Output: "result1"}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{Name: "tool2", Output: "result2"}},
				{Role: core.RoleAssistant, Content: "response"},
			},
			useToolRole: false,
			expected:    5,
			checkRoles:  true,
		},
		{
			name: "no consecutive same roles",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "user"},
				{Role: core.RoleAssistant, Content: "assistant"},
			},
			useToolRole: false,
			expected:    3,
		},
		{
			name: "single message unchanged",
			messages: []core.Message{
				{Role: core.RoleUser, Content: "hello"},
			},
			useToolRole: false,
			expected:    1,
		},
		{
			name:        "empty messages",
			messages:    []core.Message{},
			useToolRole: false,
			expected:    0,
		},
		{
			name: "three consecutive user messages",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "first", Tokens: 10},
				{Role: core.RoleUser, Content: "second", Tokens: 10},
				{Role: core.RoleUser, Content: "third", Tokens: 10},
				{Role: core.RoleAssistant, Content: "response"},
			},
			useToolRole: false,
			expected:    3,
			checkRoles:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeMessages(tt.messages, tt.useToolRole)

			if len(result) != tt.expected {
				t.Errorf("expected %d messages, got %d", tt.expected, len(result))
			}

			if tt.checkRoles && len(result) > 1 {
				for i := 1; i < len(result); i++ {
					prevRole := effectiveRole(result[i-1], tt.useToolRole)
					currRole := effectiveRole(result[i], tt.useToolRole)

					if prevRole == currRole && currRole != core.RoleTool {
						t.Errorf("found consecutive same roles at index %d: %s -> %s",
							i, prevRole, currRole)
					}
				}
			}
		})
	}
}

func TestNormalizeMessages_ContentMerging(t *testing.T) {
	messages := []core.Message{
		{Role: core.RoleUser, Content: "first", Tokens: 5},
		{Role: core.RoleUser, Content: "second", Tokens: 6},
		{Role: core.RoleUser, Content: "third", Tokens: 7},
	}

	result := normalizeMessages(messages, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	merged := result[0]

	expectedContent := "first\n\n---\n\nsecond\n\n---\n\nthird"
	if merged.Content != expectedContent {
		t.Errorf("content not merged correctly\nexpected: %q\ngot: %q",
			expectedContent, merged.Content)
	}

	expectedTokens := 5 + 6 + 7
	if merged.Tokens != expectedTokens {
		t.Errorf("expected tokens %d, got %d", expectedTokens, merged.Tokens)
	}
}

func TestNormalizeMessages_ToolCallsMerging(t *testing.T) {
	messages := []core.Message{
		{
			Role:    core.RoleAssistant,
			Content: "calling tools",
			ToolCalls: []core.ToolCall{
				{ID: "call1", Name: "tool1"},
			},
		},
		{
			Role:    core.RoleAssistant,
			Content: "more calls",
			ToolCalls: []core.ToolCall{
				{ID: "call2", Name: "tool2"},
			},
		},
	}

	result := normalizeMessages(messages, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	merged := result[0]

	if len(merged.ToolCalls) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(merged.ToolCalls))
	}
}
