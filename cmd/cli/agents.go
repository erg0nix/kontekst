package main

import (
	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

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
		lipgloss.Println(styleDim.Render("No agents found."))
		lipgloss.Println("Create an agent by adding a directory to " + styleToolName.Render("~/.kontekst/agents/"))
		return nil
	}

	printAgentsTable(agentList)
	return nil
}

func printAgentsTable(agentList []agent.AgentSummary) {
	t := table.New().
		Headers("NAME", "DISPLAY NAME", "PROMPT", "CONFIG").
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderHeader(true).
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader
			}
			return lipgloss.NewStyle().PaddingRight(2)
		})

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
