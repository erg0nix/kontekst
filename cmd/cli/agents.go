package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "List available agents",
		RunE:  runAgentsCmd,
	}

	return cmd
}

func runAgentsCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	registry := agent.NewRegistry(cfg.DataDir)
	agentList, err := registry.List()
	if err != nil {
		return err
	}

	if len(agentList) == 0 {
		fmt.Println("No agents found.")
		fmt.Println("Create an agent by adding a directory to ~/.kontekst/agents/")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISPLAY NAME\tPROMPT\tCONFIG")

	for _, agent := range agentList {
		prompt := "-"
		if agent.HasPrompt {
			prompt = "✓"
		}
		config := "-"
		if agent.HasConfig {
			config = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", agent.Name, agent.DisplayName, prompt, config)
	}

	w.Flush()
	return nil
}
