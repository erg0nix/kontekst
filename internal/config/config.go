// Package config loads and manages the server-level TOML configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// FallbackContextSize is the default context window size used when no agent-specific value is configured.
const FallbackContextSize = 4096

// FileToolsConfig holds configuration for file-based tool operations.
type FileToolsConfig struct {
	MaxSizeBytes int64 `toml:"max_size_bytes"`
}

// WebToolsConfig holds configuration for web-based tool operations.
type WebToolsConfig struct {
	TimeoutSeconds   int   `toml:"timeout_seconds"`
	MaxResponseBytes int64 `toml:"max_response_bytes"`
}

// ToolsConfig holds configuration for all tool subsystems.
type ToolsConfig struct {
	WorkingDir string          `toml:"working_dir"`
	File       FileToolsConfig `toml:"file"`
	Web        WebToolsConfig  `toml:"web"`
}

// DebugConfig holds settings for debug logging and validation.
type DebugConfig struct {
	LogRequests   bool   `toml:"log_requests"`
	LogResponses  bool   `toml:"log_responses"`
	LogDirectory  string `toml:"log_directory"`
	ValidateRoles bool   `toml:"validate_roles"`
}

// Config is the top-level server configuration loaded from config.toml.
type Config struct {
	Bind    string      `toml:"bind"`
	DataDir string      `toml:"data_dir"`
	Tools   ToolsConfig `toml:"tools"`
	Debug   DebugConfig `toml:"debug"`
}

// Default returns a Config populated with sensible default values.
func Default() Config {
	defaultDataDir := defaultDataDir()
	return Config{
		Bind:    ":50051",
		DataDir: defaultDataDir,
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
		},
	}
}

// LoadOrCreate reads the config file at path, creating it with defaults if it does not exist.
func LoadOrCreate(path string) (Config, error) {
	config := Default()

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return config, fmt.Errorf("config: create directory: %w", err)
			}

			configData, err := toml.Marshal(config)
			if err != nil {
				return config, fmt.Errorf("config: marshal defaults: %w", err)
			}

			if err := os.WriteFile(path, configData, 0o644); err != nil {
				return config, fmt.Errorf("config: write defaults: %w", err)
			}

			return config, nil
		}

		return config, fmt.Errorf("config: stat %s: %w", path, err)
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("config: read %s: %w", path, err)
	}

	if err := toml.Unmarshal(configData, &config); err != nil {
		return config, fmt.Errorf("config: parse %s: %w", path, err)
	}

	config.DataDir = expandPath(config.DataDir)
	config.Bind = strings.TrimSpace(config.Bind)

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
