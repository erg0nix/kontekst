package agents

import (
	"os"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/pelletier/go-toml/v2"
)

type ProviderTOML struct {
	Endpoint           string `toml:"endpoint"`
	Model              string `toml:"model"`
	HTTPTimeoutSeconds int    `toml:"http_timeout_seconds"`
}

type ProviderConfig struct {
	Endpoint    string
	Model       string
	HTTPTimeout time.Duration
}

type AgentConfig struct {
	Name         string
	DisplayName  string
	SystemPrompt string
	ContextSize  int
	Provider     ProviderConfig
	Sampling     *core.SamplingConfig
	ToolRole     bool
}

type AgentTOML struct {
	Name        string               `toml:"name"`
	ContextSize int                  `toml:"context_size"`
	Provider    ProviderTOML         `toml:"provider"`
	Sampling    *core.SamplingConfig `toml:"sampling"`
	ToolRole    bool                 `toml:"tool_role"`
}

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
