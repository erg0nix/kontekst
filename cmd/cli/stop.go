package main

import (
	"context"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/acp"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the kontekst server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			serverOverride, _ := cmd.Flags().GetString("server")
			cfg, _ := loadConfig(configPath)
			serverAddr := resolveServer(serverOverride, cfg)

			client, err := dialServer(serverAddr, acp.ClientCallbacks{})
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := client.Shutdown(ctx); err != nil {
				printServerNotRunning(serverAddr, err)
				return err
			}

			lipgloss.Println(styleDim.Render("shutting down"))
			return nil
		},
	}

	return cmd
}
