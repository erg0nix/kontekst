package cli

import (
	"fmt"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/spf13/cobra"
)

type App struct {
	Config     config.Config
	ConfigPath string
	ServerAddr string
}

func newApp(cmd *cobra.Command) (*App, error) {
	configPath, _ := cmd.Flags().GetString("config")
	serverOverride, _ := cmd.Flags().GetString("server")

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &App{
		Config:     cfg,
		ConfigPath: configPath,
		ServerAddr: resolveServer(serverOverride, cfg),
	}, nil
}
