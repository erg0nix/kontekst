package provider

import "github.com/erg0nix/kontekst/internal/core"

// Response holds the parsed response from an LLM completion request.
type Response struct {
	Content   string
	Reasoning string
	ToolCalls []core.ToolCall
	Usage     *Usage
}

// Usage tracks token consumption for a single LLM request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
