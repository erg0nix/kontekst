package agent

import (
	"os"
	"path/filepath"
	"strings"

	agentcfg "github.com/erg0nix/kontekst/internal/config/agents"
)

type Registry struct {
	AgentsDir string
	ModelDir  string
}

func NewRegistry(dataDir string, modelDir string) *Registry {
	return &Registry{
		AgentsDir: filepath.Join(dataDir, "agents"),
		ModelDir:  modelDir,
	}
}

type AgentSummary struct {
	Name        string
	DisplayName string
	HasPrompt   bool
	HasConfig   bool
}

func (r *Registry) List() ([]AgentSummary, error) {
	entries, err := os.ReadDir(r.AgentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []AgentSummary
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
			cfg, err := agentcfg.LoadTOML(configPath)
			if err == nil && cfg != nil && cfg.Name != "" {
				displayName = cfg.Name
			}
		}

		agents = append(agents, AgentSummary{
			Name:        name,
			DisplayName: displayName,
			HasPrompt:   hasPrompt,
			HasConfig:   hasConfig,
		})
	}

	return agents, nil
}

func (r *Registry) Load(name string) (*agentcfg.AgentConfig, error) {
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
		return nil, &AgentNotFoundError{Name: name, Available: names}
	}

	cfg := &agentcfg.AgentConfig{
		Name:        name,
		DisplayName: name,
	}

	if hasConfig {
		tomlCfg, err := agentcfg.LoadTOML(configPath)
		if err != nil {
			return nil, &AgentConfigError{Name: name, Err: err}
		}
		if tomlCfg != nil {
			if tomlCfg.Name != "" {
				cfg.DisplayName = tomlCfg.Name
			}
			if tomlCfg.Model != "" {
				if filepath.IsAbs(tomlCfg.Model) {
					cfg.Model = tomlCfg.Model
				} else if r.ModelDir != "" {
					cfg.Model = filepath.Join(r.ModelDir, tomlCfg.Model)
				} else {
					cfg.Model = tomlCfg.Model
				}
			}
			cfg.Sampling = tomlCfg.Sampling
		}
	}

	if hasPrompt {
		prompt, err := agentcfg.LoadPrompt(promptPath)
		if err != nil {
			return nil, &AgentConfigError{Name: name, Err: err}
		}
		cfg.SystemPrompt = prompt
	}

	return cfg, nil
}

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

type AgentNotFoundError struct {
	Name      string
	Available []string
}

func (e *AgentNotFoundError) Error() string {
	msg := "agent not found: " + e.Name
	if len(e.Available) > 0 {
		msg += "; available: " + strings.Join(e.Available, ", ")
	}
	return msg
}

type AgentConfigError struct {
	Name string
	Err  error
}

func (e *AgentConfigError) Error() string {
	return "invalid config for agent " + e.Name + ": " + e.Err.Error()
}
