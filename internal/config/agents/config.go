package agents

import (
	"os"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/pelletier/go-toml/v2"
)

type AgentConfig struct {
	Name         string
	DisplayName  string
	SystemPrompt string
	Model        string
	Sampling     *core.SamplingConfig
	ToolRole     bool
}

type AgentTOML struct {
	Name     string               `toml:"name"`
	Model    string               `toml:"model"`
	Sampling *core.SamplingConfig `toml:"sampling"`
	ToolRole bool                 `toml:"tool_role"`
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
