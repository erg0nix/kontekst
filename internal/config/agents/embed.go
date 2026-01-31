package agents

import (
	_ "embed"
)

//go:embed prompts/system-prompt.md
var DefaultSystemPrompt string
