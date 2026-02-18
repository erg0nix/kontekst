package providers

import (
	"github.com/erg0nix/kontekst/internal/core"
)

// Provider defines the interface for LLM backends that generate chat completions and count tokens.
type Provider interface {
	GenerateChat(
		messages []core.Message,
		tools []core.ToolDef,
		sampling *core.SamplingConfig,
		model string,
		useToolRole bool,
	) (core.ChatResponse, error)
	CountTokens(text string) (int, error)
}
