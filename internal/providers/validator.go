package providers

import (
	"fmt"

	"github.com/erg0nix/kontekst/internal/core"
)

func validateRoleAlternation(messages []core.Message, useToolRole bool) error {
	if len(messages) == 0 {
		return nil
	}

	if messages[0].Role != core.RoleSystem {
		return fmt.Errorf("first message must be system role, got: %s", messages[0].Role)
	}

	var prevRole core.Role
	expectingToolResult := false

	for i := 1; i < len(messages); i++ {
		msg := messages[i]
		actualRole := effectiveRole(msg, useToolRole)

		if actualRole == prevRole && actualRole != core.RoleTool {
			return fmt.Errorf("consecutive %s messages at index %d and %d", actualRole, i-1, i)
		}

		if msg.Role == core.RoleTool || msg.ToolResult != nil {
			if !expectingToolResult {
				return fmt.Errorf("tool result at index %d without preceding assistant tool calls", i)
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
