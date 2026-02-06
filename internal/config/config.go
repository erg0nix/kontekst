package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type FileToolsConfig struct {
	MaxSizeBytes int64 `toml:"max_size_bytes"`
}

type WebToolsConfig struct {
	TimeoutSeconds   int   `toml:"timeout_seconds"`
	MaxResponseBytes int64 `toml:"max_response_bytes"`
}

type ToolsConfig struct {
	WorkingDir string          `toml:"working_dir"`
	File       FileToolsConfig `toml:"file"`
	Web        WebToolsConfig  `toml:"web"`
}

type DebugConfig struct {
	LogRequests   bool   `toml:"log_requests"`
	LogResponses  bool   `toml:"log_responses"`
	LogDirectory  string `toml:"log_directory"`
	ValidateRoles bool   `toml:"validate_roles"`
	DumpOnError   bool   `toml:"dump_on_error"`
}

type Config struct {
	Bind        string      `toml:"bind"`
	Endpoint    string      `toml:"endpoint"`
	ModelDir    string      `toml:"model_dir"`
	ContextSize int         `toml:"context_size"`
	GPULayers   int         `toml:"gpu_layers"`
	DataDir     string      `toml:"data_dir"`
	Tools       ToolsConfig `toml:"tools"`
	Debug       DebugConfig `toml:"debug"`
}

func Default() Config {
	defaultDataDir := defaultDataDir()
	return Config{
		Bind:        ":50051",
		Endpoint:    "http://127.0.0.1:8080",
		ModelDir:    defaultModelsDir(),
		ContextSize: 4096,
		GPULayers:   0,
		DataDir:     defaultDataDir,
		Tools: ToolsConfig{
			WorkingDir: "",
			File: FileToolsConfig{
				MaxSizeBytes: 10 * 1024 * 1024,
			},
			Web: WebToolsConfig{
				TimeoutSeconds:   30,
				MaxResponseBytes: 5 * 1024 * 1024,
			},
		},
		Debug: DebugConfig{
			LogRequests:   false,
			LogResponses:  false,
			LogDirectory:  filepath.Join(defaultDataDir, "debug"),
			ValidateRoles: true,
			DumpOnError:   true,
		},
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
