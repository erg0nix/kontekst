package context

import "github.com/erg0nix/kontekst/internal/core"

type ContextWindow interface {
	AddMessage(msg core.Message) error
	BuildContext(countTokens func(string) (int, error)) ([]core.Message, error)
	RenderUserMessage(prompt string) (string, error)
	AddToolResult(result core.ToolResult) error
	SetAgentSystemPrompt(prompt string)
	SetActiveSkill(skill *core.SkillMetadata)
	ActiveSkill() *core.SkillMetadata
}

type ContextService interface {
	NewWindow(sessionID core.SessionID) (ContextWindow, error)
}
