package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Bind           string `toml:"bind"`
	Endpoint       string `toml:"endpoint"`
	Model          string `toml:"model"`
	ModelDir       string `toml:"model_dir"`
	LlamaServerBin string `toml:"llama_server_bin"`
	ContextSize    int    `toml:"context_size"`
	GPULayers      int    `toml:"gpu_layers"`
	MaxTokens      int    `toml:"max_tokens"`
	DataDir        string `toml:"data_dir"`
}

func Default() Config {
	return Config{
		Bind:        ":50051",
		Endpoint:    "http://127.0.0.1:8080",
		Model:       filepath.Join(defaultModelsDir(), "gpt-oss-20b-Q4_K_M.gguf"),
		ModelDir:    defaultModelsDir(),
		ContextSize: 4096,
		GPULayers:   0,
		MaxTokens:   4096,
		DataDir:     defaultDataDir(),
	}
}

func LoadOrCreate(path string) (Config, error) {
	config := Default()

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return config, err
			}

			configData, err := toml.Marshal(config)
			if err != nil {
				return config, err
			}

			if err := os.WriteFile(path, configData, 0o644); err != nil {
				return config, err
			}

			return config, nil
		}

		return config, err
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	if err := toml.Unmarshal(configData, &config); err != nil {
		return config, err
	}

	config.DataDir = expandPath(config.DataDir)
	config.ModelDir = expandPath(config.ModelDir)
	config.Model = expandPath(config.Model)
	config.LlamaServerBin = expandPath(config.LlamaServerBin)
	config.Endpoint = strings.TrimSpace(config.Endpoint)
	config.Bind = strings.TrimSpace(config.Bind)

	if config.Endpoint == "" {
		return config, errors.New("endpoint is required")
	}

	if config.Bind == "" {
		config.Bind = ":50051"
	}

	return config, nil
}

func defaultDataDir() string {
	homeDir, _ := os.UserHomeDir()

	if homeDir == "" {
		return ".kontekst"
	}

	return filepath.Join(homeDir, ".kontekst")
}

func defaultModelsDir() string {
	homeDir, _ := os.UserHomeDir()

	if homeDir == "" {
		return "models"
	}

	return filepath.Join(homeDir, "models")
}

func expandPath(path string) string {
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "~") {
		homeDir, _ := os.UserHomeDir()

		if homeDir != "" {
			trimmed := strings.TrimPrefix(path, "~")
			trimmed = strings.TrimPrefix(trimmed, string(os.PathSeparator))

			return filepath.Join(homeDir, trimmed)
		}
	}

	return path
}
