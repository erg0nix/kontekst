package providers

import (
	"fmt"

	"github.com/erg0nix/kontekst/internal/core"
)

type validationError struct {
	index        int
	currentRole  core.Role
	previousRole core.Role
	Message      string
}

func (e *validationError) Error() string {
	return e.Message
}

func validateRoleAlternation(messages []core.Message, useToolRole bool) error {
	if len(messages) == 0 {
		return nil
	}

	if messages[0].Role != core.RoleSystem {
		return &validationError{
			index:   0,
			Message: fmt.Sprintf("first message must be system role, got: %s", messages[0].Role),
		}
	}

	var prevRole core.Role
	expectingToolResult := false

	for i := 1; i < len(messages); i++ {
		msg := messages[i]
		actualRole := effectiveRole(msg, useToolRole)

		if actualRole == prevRole && actualRole != core.RoleTool {
			return &validationError{
				index:        i,
				currentRole:  actualRole,
				previousRole: prevRole,
				Message:      fmt.Sprintf("consecutive %s messages at index %d and %d", actualRole, i-1, i),
			}
		}

		if msg.Role == core.RoleTool || msg.ToolResult != nil {
			if !expectingToolResult {
				return &validationError{
					index:   i,
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
