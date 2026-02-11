package main

import (
	"context"
	"time"

	"github.com/erg0nix/kontekst/internal/acp"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "Show server status",
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

			statusResp, err := client.Status(ctx)
			if err != nil {
				printServerNotRunning(serverAddr, err)
				return err
			}

			printStatus(serverAddr, statusResp)
			return nil
		},
	}

	return cmd
}
