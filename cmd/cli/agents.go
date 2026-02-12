package main

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/sessions"
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
		lipgloss.Println(styleDim.Render("No agents found."))
		lipgloss.Println("Create an agent by adding a directory to " + styleToolName.Render("~/.kontekst/agents/"))
		return nil
	}

	if !isInteractive() {
		printAgentsTable(agentList)
		return nil
	}

	return pickAgent(cfg.DataDir, agentList)
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

func pickAgent(dataDir string, agentList []agent.AgentSummary) error {
	currentAgent := ""
	sessionID := loadActiveSession(dataDir)
	if sessionID != "" {
		svc := &sessions.FileSessionService{BaseDir: dataDir}
		if name, err := svc.GetDefaultAgent(core.SessionID(sessionID)); err == nil {
			currentAgent = name
		}
	}

	var opts []huh.Option[string]
	for _, a := range agentList {
		label := a.DisplayName
		if label != a.Name {
			label = fmt.Sprintf("%s (%s)", a.DisplayName, a.Name)
		}
		if a.Name == currentAgent {
			label = "* " + label
		}
		opt := huh.NewOption(label, a.Name)
		if a.Name == currentAgent {
			opt = opt.Selected(true)
		}
		opts = append(opts, opt)
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Pick an agent").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}

	if sessionID == "" {
		lipgloss.Printf("%s %s\n", styleSuccess.Render("Selected"), selected)
		return nil
	}

	svc := &sessions.FileSessionService{BaseDir: dataDir}
	if err := svc.SetDefaultAgent(core.SessionID(sessionID), selected); err != nil {
		return fmt.Errorf("set default agent: %w", err)
	}

	lipgloss.Printf("%s default agent for session %s to %q\n",
		styleSuccess.Render("Set"), sessionID, selected)
	return nil
}
