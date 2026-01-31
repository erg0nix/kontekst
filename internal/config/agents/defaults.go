package agents

import (
	"os"
	"path/filepath"
)

const DefaultAgentName = "default"

const defaultAgentConfigTOML = `name = "Default Assistant"
model = "gpt-oss-20b-Q4_K_M.gguf"

[sampling]
temperature = 0.7
top_p = 0.9
top_k = 40
repeat_penalty = 1.1
max_tokens = 4096
`

const defaultAgentPromptMD = `You are a helpful assistant.
`

func EnsureDefault(baseDir string) error {
	agentDir := filepath.Join(baseDir, "agents", DefaultAgentName)

	if _, err := os.Stat(agentDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(agentDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(defaultAgentConfigTOML), 0o644); err != nil {
		return err
	}

	promptPath := filepath.Join(agentDir, "agent.md")
	if err := os.WriteFile(promptPath, []byte(defaultAgentPromptMD), 0o644); err != nil {
		return err
	}

	return nil
}
