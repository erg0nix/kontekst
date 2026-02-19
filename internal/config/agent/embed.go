package agent

import (
	_ "embed"
)

// DefaultSystemPrompt is the embedded system prompt for the default assistant agent.
//
//go:embed prompts/system-prompt.md
var DefaultSystemPrompt string

// CoderSystemPrompt is the embedded system prompt for the coder agent.
//
//go:embed prompts/coder-prompt.md
var CoderSystemPrompt string

// FantasySystemPrompt is the embedded system prompt for the fantasy writer agent.
//
//go:embed prompts/fantasy-prompt.md
var FantasySystemPrompt string

// InitSystemPrompt is the embedded system prompt for the project initializer agent.
//
//go:embed prompts/init-prompt.md
var InitSystemPrompt string
