package agents

import (
	_ "embed"
)

//go:embed prompts/system-prompt.md
var DefaultSystemPrompt string

//go:embed prompts/coder-prompt.md
var CoderSystemPrompt string

//go:embed prompts/fantasy-prompt.md
var FantasySystemPrompt string
