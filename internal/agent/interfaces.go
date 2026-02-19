package agent

import (
	"github.com/erg0nix/kontekst/internal/conversation"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/provider"
)

// ConversationFactory creates conversation windows for sessions.
type ConversationFactory interface {
	NewWindow(sessionID core.SessionID) (ConversationWindow, error)
}

// ConversationWindow manages the message history and system prompt for a session.
type ConversationWindow interface {
	SystemContent() string
	StartRun(params conversation.BudgetParams) error
	CompleteRun()
	AddMessage(msg core.Message) error
	BuildContext() ([]core.Message, error)
	SetAgentSystemPrompt(prompt string)
	SetActiveSkill(skill *core.SkillMetadata)
	ActiveSkill() *core.SkillMetadata
	Snapshot() conversation.Snapshot
}

// LLM generates chat completions and counts tokens.
type LLM interface {
	GenerateChat(
		messages []core.Message,
		tools []core.ToolDef,
		sampling *core.SamplingConfig,
		model string,
		useToolRole bool,
	) (provider.Response, error)
	CountTokens(text string) (int, error)
}

// SessionStore creates and ensures session existence.
type SessionStore interface {
	Create() (core.SessionID, string, error)
	Ensure(sessionID core.SessionID) (string, error)
}
