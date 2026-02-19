package agent

import (
	"os"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/pelletier/go-toml/v2"
)

// ProviderTOML is the TOML-serializable representation of an LLM provider configuration.
type ProviderTOML struct {
	Endpoint           string `toml:"endpoint"`
	Model              string `toml:"model"`
	HTTPTimeoutSeconds int    `toml:"http_timeout_seconds"`
}

// ProviderConfig holds the resolved provider settings used at runtime.
type ProviderConfig struct {
	Endpoint    string
	Model       string
	HTTPTimeout time.Duration
}

// AgentConfig is the fully resolved configuration for an agent, ready for use by the agent loop.
type AgentConfig struct {
	Name         string
	DisplayName  string
	SystemPrompt string
	ContextSize  int
	Provider     ProviderConfig
	Sampling     *core.SamplingConfig
	ToolRole     bool
}

// AgentTOML is the TOML-serializable representation of an agent's configuration file.
type AgentTOML struct {
	Name        string               `toml:"name"`
	ContextSize int                  `toml:"context_size"`
	Provider    ProviderTOML         `toml:"provider"`
	Sampling    *core.SamplingConfig `toml:"sampling"`
	ToolRole    bool                 `toml:"tool_role"`
}

// LoadTOML reads and parses an agent TOML config file, returning nil if the file does not exist.
func LoadTOML(path string) (*AgentTOML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg AgentTOML
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadPrompt reads a system prompt file, returning an empty string if the file does not exist.
func LoadPrompt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(data), nil
}
