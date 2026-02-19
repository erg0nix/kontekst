package agent

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
)

// Registry discovers and loads agent configurations from the agents directory.
type Registry struct {
	AgentsDir string
}

// NewRegistry creates a Registry that looks for agents under dataDir/agents.
func NewRegistry(dataDir string) *Registry {
	return &Registry{
		AgentsDir: filepath.Join(dataDir, "agents"),
	}
}

// Summary holds metadata about a registered agent for listing purposes.
type Summary struct {
	Name        string
	DisplayName string
	HasPrompt   bool
	HasConfig   bool
}

// List returns summaries of all agents found in the agents directory.
func (r *Registry) List() ([]Summary, error) {
	entries, err := os.ReadDir(r.AgentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []Summary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		agentDir := filepath.Join(r.AgentsDir, name)
		configPath := filepath.Join(agentDir, "config.toml")
		promptPath := filepath.Join(agentDir, "agent.md")

		hasConfig := fileExists(configPath)
		hasPrompt := fileExists(promptPath)

		if !hasConfig && !hasPrompt {
			continue
		}

		displayName := name
		if hasConfig {
			cfg, err := agentConfig.LoadTOML(configPath)
			if err == nil && cfg != nil && cfg.Name != "" {
				displayName = cfg.Name
			}
		}

		agents = append(agents, Summary{
			Name:        name,
			DisplayName: displayName,
			HasPrompt:   hasPrompt,
			HasConfig:   hasConfig,
		})
	}

	return agents, nil
}

// Load reads and returns the full agent configuration for the named agent.
func (r *Registry) Load(name string) (*agentConfig.AgentConfig, error) {
	agentDir := filepath.Join(r.AgentsDir, name)
	configPath := filepath.Join(agentDir, "config.toml")
	promptPath := filepath.Join(agentDir, "agent.md")

	hasConfig := fileExists(configPath)
	hasPrompt := fileExists(promptPath)

	if !hasConfig && !hasPrompt {
		available, _ := r.List()
		var names []string
		for _, a := range available {
			names = append(names, a.Name)
		}
		return nil, &NotFoundError{Name: name, Available: names}
	}

	cfg := &agentConfig.AgentConfig{
		Name:        name,
		DisplayName: name,
	}

	if hasConfig {
		tomlCfg, err := agentConfig.LoadTOML(configPath)
		if err != nil {
			return nil, &ConfigError{Name: name, Err: err}
		}
		if tomlCfg != nil {
			if tomlCfg.Name != "" {
				cfg.DisplayName = tomlCfg.Name
			}

			cfg.Provider = agentConfig.ProviderConfig{
				Endpoint: tomlCfg.Provider.Endpoint,
				Model:    tomlCfg.Provider.Model,
			}
			if tomlCfg.Provider.HTTPTimeoutSeconds > 0 {
				cfg.Provider.HTTPTimeout = time.Duration(tomlCfg.Provider.HTTPTimeoutSeconds) * time.Second
			} else {
				cfg.Provider.HTTPTimeout = 300 * time.Second
			}

			cfg.ContextSize = tomlCfg.ContextSize
			cfg.Sampling = tomlCfg.Sampling
			cfg.ToolRole = tomlCfg.ToolRole
		}
	}

	if hasPrompt {
		prompt, err := agentConfig.LoadPrompt(promptPath)
		if err != nil {
			return nil, &ConfigError{Name: name, Err: err}
		}
		cfg.SystemPrompt = prompt
	}

	return cfg, nil
}

// Exists reports whether an agent with the given name has a config or prompt file.
func (r *Registry) Exists(name string) bool {
	agentDir := filepath.Join(r.AgentsDir, name)
	configPath := filepath.Join(agentDir, "config.toml")
	promptPath := filepath.Join(agentDir, "agent.md")

	return fileExists(configPath) || fileExists(promptPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// NotFoundError is returned when a requested agent does not exist in the registry.
type NotFoundError struct {
	Name      string
	Available []string
}

// Error returns a message identifying the missing agent and listing available alternatives.
func (e *NotFoundError) Error() string {
	msg := "agent not found: " + e.Name
	if len(e.Available) > 0 {
		msg += "; available: " + strings.Join(e.Available, ", ")
	}
	return msg
}

// ConfigError is returned when an agent's configuration file cannot be loaded or parsed.
type ConfigError struct {
	Name string
	Err  error
}

// Error returns a message identifying the agent and the underlying configuration error.
func (e *ConfigError) Error() string {
	return "invalid config for agent " + e.Name + ": " + e.Err.Error()
}

// Unwrap returns the underlying error that caused the configuration failure.
func (e *ConfigError) Unwrap() error {
	return e.Err
}
