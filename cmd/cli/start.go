package main

import (
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the kontekst server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			foreground, _ := cmd.Flags().GetBool("foreground")
			cfg, _ := loadConfig(configPath)
			return startServer(cfg, configPath, foreground)
		},
	}

	cmd.Flags().Bool("foreground", false, "run server in foreground")

	return cmd
}
