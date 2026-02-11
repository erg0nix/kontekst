package main

import (
	"log/slog"
	"os"

	"github.com/erg0nix/kontekst/internal/acp"
	"github.com/erg0nix/kontekst/internal/config"

	"github.com/spf13/cobra"
)

func newACPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run as ACP agent over stdio (for editors)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

			cfg, _ := loadConfig(configPath)
			cfg.Debug = config.LoadDebugConfigFromEnv(cfg.Debug)

			services := setupServices(cfg)
			handler := acp.NewHandler(services.Runner, services.Agents, services.Skills)
			conn := handler.Serve(os.Stdout, os.Stdin)

			<-conn.Done()
			return nil
		},
	}

	return cmd
}
