package main

import (
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the kontekst daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			daemonPath, _ := cmd.Flags().GetString("daemon-bin")
			foreground, _ := cmd.Flags().GetBool("foreground")
			config, _ := loadConfig(configPath)
			return startDaemon(config, configPath, daemonPath, foreground)
		},
	}

	cmd.Flags().String("daemon-bin", "", "path to daemon binary")
	cmd.Flags().Bool("foreground", false, "run daemon in foreground")

	return cmd
}
