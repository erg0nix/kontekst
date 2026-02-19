// Package cli implements the Cobra command tree for the kontekst CLI.
package cli

import (
	"fmt"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List available agents",
		RunE:  runAgentsCmd,
	}
}

func runAgentsCmd(cmd *cobra.Command, _ []string) error {
	app, err := newApp(cmd)
	if err != nil {
		return err
	}

	registry := agent.NewRegistry(app.Config.DataDir)
	agentList, err := registry.List()
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}

	if len(agentList) == 0 {
		lipgloss.Println(styleDim.Render("No agents found."))
		lipgloss.Println("Create an agent by adding a directory to " + styleToolName.Render("~/.kontekst/agents/"))
		return nil
	}

	printAgentsTable(agentList)
	return nil
}

func printAgentsTable(agentList []agent.Summary) {
	t := newTable("NAME", "DISPLAY NAME", "PROMPT", "CONFIG")

	for _, a := range agentList {
		prompt := styleDim.Render("-")
		if a.HasPrompt {
			prompt = styleSuccess.Render("✓")
		}
		config := styleDim.Render("-")
		if a.HasConfig {
			config = styleSuccess.Render("✓")
		}
		t.Row(a.Name, a.DisplayName, prompt, config)
	}

	lipgloss.Println(t.Render())
}
