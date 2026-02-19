package agent

import (
	"os"
	"path/filepath"
)

// DefaultAgentName is the name of the default agent used when none is specified.
const DefaultAgentName = "default"

// InitAgentName is the name of the agent used for project initialization.
const InitAgentName = "init"

type bundledAgent struct {
	name   string
	config string
	prompt string
}

var bundledAgents = []bundledAgent{
	{name: "default", prompt: DefaultSystemPrompt, config: defaultConfig},
	{name: "coder", prompt: CoderSystemPrompt, config: coderConfig},
	{name: "fantasy", prompt: FantasySystemPrompt, config: fantasyConfig},
	{name: InitAgentName, prompt: InitSystemPrompt, config: initConfig},
}

// EnsureDefaults creates bundled agent configurations under baseDir if they do not already exist.
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
