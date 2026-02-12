package agents

import (
	"os"
	"path/filepath"
)

const DefaultAgentName = "default"

type bundledAgent struct {
	name   string
	config string
	prompt string
}

var bundledAgents = []bundledAgent{
	{
		name:   "default",
		prompt: DefaultSystemPrompt,
		config: `name = "Default Assistant"
context_size = 4096
tool_role = false

[provider]
endpoint = "http://127.0.0.1:8080"
model = "gpt-oss-20b-Q4_K_M.gguf"

[sampling]
temperature = 0.7
top_p = 0.9
top_k = 40
repeat_penalty = 1.1
max_tokens = 4096
`,
	},
	{
		name:   "coder",
		prompt: CoderSystemPrompt,
		config: `name = "Coder"
context_size = 4096
tool_role = false

[provider]
endpoint = "http://127.0.0.1:8080"
model = "gpt-oss-20b-Q4_K_M.gguf"

[sampling]
temperature = 0.3
top_p = 0.9
top_k = 40
repeat_penalty = 1.1
max_tokens = 4096
`,
	},
	{
		name:   "fantasy",
		prompt: FantasySystemPrompt,
		config: `name = "Fantasy Writer"
context_size = 4096
tool_role = false

[provider]
endpoint = "http://127.0.0.1:8080"
model = "gpt-oss-20b-Q4_K_M.gguf"

[sampling]
temperature = 0.9
top_p = 0.9
top_k = 60
repeat_penalty = 1.1
max_tokens = 4096
`,
	},
}

func EnsureDefaults(baseDir string) error {
	for _, a := range bundledAgents {
		if err := ensureAgent(baseDir, a); err != nil {
			return err
		}
	}
	return nil
}

func ensureAgent(baseDir string, a bundledAgent) error {
	agentDir := filepath.Join(baseDir, "agents", a.name)

	if _, err := os.Stat(agentDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(agentDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(a.config), 0o644); err != nil {
		return err
	}

	promptPath := filepath.Join(agentDir, "agent.md")
	return os.WriteFile(promptPath, []byte(a.prompt), 0o644)
}
