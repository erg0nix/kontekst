package providers

import (
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestValidateRoleAlternation(t *testing.T) {
	tests := []struct {
		name        string
		messages    []core.Message
		useToolRole bool
		expectError bool
		errorSubstr string
	}{
		{
			name: "valid simple conversation",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "hello"},
				{Role: core.RoleAssistant, Content: "hi"},
			},
			useToolRole: false,
			expectError: false,
		},
		{
			name: "invalid consecutive user messages",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "first"},
				{Role: core.RoleUser, Content: "second"},
			},
			useToolRole: false,
			expectError: true,
			errorSubstr: "consecutive user",
		},
		{
			name: "invalid consecutive assistant messages",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "question"},
				{Role: core.RoleAssistant, Content: "answer"},
				{Role: core.RoleAssistant, Content: "more"},
			},
			useToolRole: false,
			expectError: true,
			errorSubstr: "consecutive assistant",
		},
		{
			name: "valid tool calls with native tool role",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "read file.txt"},
				{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{{ID: "1", Name: "read_file"}}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "1", Name: "read_file", Output: "content"}},
				{Role: core.RoleAssistant, Content: "the file contains..."},
			},
			useToolRole: true,
			expectError: false,
		},
		{
			name: "valid tool calls without native tool role (becomes user)",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "read file.txt"},
				{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{{ID: "1", Name: "read_file"}}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "1", Name: "read_file", Output: "content"}},
				{Role: core.RoleAssistant, Content: "the file contains..."},
			},
			useToolRole: false,
			expectError: false,
		},
		{
			name: "invalid tool result without tool calls",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "question"},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{Name: "read_file", Output: "content"}},
			},
			useToolRole: true,
			expectError: true,
			errorSubstr: "without preceding",
		},
		{
			name: "valid multiple tool results with native tool role",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "read files"},
				{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{
					{ID: "1", Name: "read_file"},
					{ID: "2", Name: "read_file"},
				}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "1", Name: "read_file", Output: "content1"}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "2", Name: "read_file", Output: "content2"}},
				{Role: core.RoleAssistant, Content: "both files contain..."},
			},
			useToolRole: true,
			expectError: false,
		},
		{
			name: "invalid multiple tool results without native tool role",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "read files"},
				{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{
					{ID: "1", Name: "read_file"},
					{ID: "2", Name: "read_file"},
				}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "1", Name: "read_file", Output: "content1"}},
				{Role: core.RoleTool, ToolResult: &core.ToolResult{CallID: "2", Name: "read_file", Output: "content2"}},
				{Role: core.RoleAssistant, Content: "both files contain..."},
			},
			useToolRole: false,
			expectError: true,
			errorSubstr: "consecutive user",
		},
		{
			name:        "empty message list",
			messages:    []core.Message{},
			useToolRole: false,
			expectError: false,
		},
		{
			name: "first message not system",
			messages: []core.Message{
				{Role: core.RoleUser, Content: "hello"},
			},
			useToolRole: false,
			expectError: true,
			errorSubstr: "first message must be system",
		},
		{
			name: "valid alternating conversation",
			messages: []core.Message{
				{Role: core.RoleSystem, Content: "system"},
				{Role: core.RoleUser, Content: "question 1"},
				{Role: core.RoleAssistant, Content: "answer 1"},
				{Role: core.RoleUser, Content: "question 2"},
				{Role: core.RoleAssistant, Content: "answer 2"},
			},
			useToolRole: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoleAlternation(tt.messages, tt.useToolRole)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorSubstr != "" {
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorSubstr)
				}
			}
		})
	}
}
