package providers

import (
	"fmt"

	"github.com/erg0nix/kontekst/internal/core"
)

type RoleValidator struct{}

type ValidationError struct {
	Index        int
	CurrentRole  core.Role
	PreviousRole core.Role
	Message      string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewRoleValidator() *RoleValidator {
	return &RoleValidator{}
}

func (v *RoleValidator) Validate(messages []core.Message, useToolRole bool) error {
	if len(messages) == 0 {
		return nil
	}

	if messages[0].Role != core.RoleSystem {
		return &ValidationError{
			Index:   0,
			Message: fmt.Sprintf("first message must be system role, got: %s", messages[0].Role),
		}
	}

	var prevRole core.Role
	expectingToolResult := false

	for i := 1; i < len(messages); i++ {
		msg := messages[i]
		actualRole := msg.Role

		if !useToolRole && msg.ToolResult != nil {
			actualRole = core.RoleUser
		}

		if actualRole == prevRole && actualRole != core.RoleTool {
			return &ValidationError{
				Index:        i,
				CurrentRole:  actualRole,
				PreviousRole: prevRole,
				Message:      fmt.Sprintf("consecutive %s messages at index %d and %d", actualRole, i-1, i),
			}
		}

		if msg.Role == core.RoleTool || msg.ToolResult != nil {
			if !expectingToolResult {
				return &ValidationError{
					Index:   i,
					Message: fmt.Sprintf("tool result at index %d without preceding assistant tool calls", i),
				}
			}
		}

		if msg.Role == core.RoleAssistant && len(msg.ToolCalls) > 0 {
			expectingToolResult = true
		} else if msg.Role == core.RoleTool || msg.ToolResult != nil {
			expectingToolResult = true
		} else {
			expectingToolResult = false
		}

		prevRole = actualRole
	}

	return nil
}
