package providers

import (
	"github.com/erg0nix/kontekst/internal/core"
)

func normalizeMessages(messages []core.Message, useToolRole bool) []core.Message {
	if len(messages) <= 1 {
		return messages
	}

	result := []core.Message{messages[0]}

	for i := 1; i < len(messages); i++ {
		current := messages[i]
		previous := &result[len(result)-1]

		if shouldMerge(current, *previous, useToolRole) {
			mergeMessages(previous, current)
		} else {
			result = append(result, current)
		}
	}

	return result
}

func shouldMerge(current, previous core.Message, useToolRole bool) bool {
	currentRole := effectiveRole(current, useToolRole)
	previousRole := effectiveRole(previous, useToolRole)

	if useToolRole && currentRole == core.RoleTool {
		return false
	}

	return currentRole == previousRole
}

func effectiveRole(msg core.Message, useToolRole bool) core.Role {
	if !useToolRole && msg.ToolResult != nil {
		return core.RoleUser
	}
	return msg.Role
}

func mergeMessages(target *core.Message, source core.Message) {
	if target.Content != "" && source.Content != "" {
		target.Content += "\n\n---\n\n" + source.Content
	} else {
		target.Content += source.Content
	}

	target.Tokens += source.Tokens

	if len(source.ToolCalls) > 0 {
		target.ToolCalls = append(target.ToolCalls, source.ToolCalls...)
	}
}
