package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/spf13/cobra"
)

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session management commands",
	}

	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionShowCmd())
	cmd.AddCommand(newSessionDeleteCmd())

	return cmd
}

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
		Args:  cobra.NoArgs,
		RunE:  runSessionListCmd,
	}
}

func runSessionListCmd(cmd *cobra.Command, _ []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	svc := &sessions.FileSessionService{BaseDir: cfg.DataDir}
	list, err := svc.List()
	if err != nil {
		return err
	}

	if len(list) == 0 {
		lipgloss.Println(styleDim.Render("No sessions found."))
		return nil
	}

	activeID := loadActiveSession(cfg.DataDir)

	if !isInteractive() {
		printSessionsTable(list, activeID)
		return nil
	}

	return pickSession(cfg.DataDir, list, activeID)
}

func printSessionsTable(list []sessions.SessionInfo, activeID string) {
	t := table.New().
		Headers("", "SESSION ID", "AGENT", "MESSAGES", "SIZE", "MODIFIED").
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

	for _, info := range list {
		marker := " "
		id := string(info.ID)
		if id == activeID {
			marker = styleActive.Render("*")
			id = styleActive.Render(id)
		}

		agentName := info.DefaultAgent
		if agentName == "" {
			agentName = "default"
		}

		t.Row(marker, id, agentName,
			fmt.Sprintf("%d", info.MessageCount),
			formatSize(info.FileSize),
			formatTime(info.ModifiedAt))
	}

	lipgloss.Println(t.Render())
}

func pickSession(dataDir string, list []sessions.SessionInfo, activeID string) error {
	var opts []huh.Option[string]
	for _, info := range list {
		id := string(info.ID)
		label := id
		if id == activeID {
			label = "* " + id
		}

		agentName := info.DefaultAgent
		if agentName == "" {
			agentName = "default"
		}
		desc := fmt.Sprintf("agent:%s msgs:%d %s", agentName, info.MessageCount, formatTime(info.ModifiedAt))

		opt := huh.NewOption(label, id)
		opt.Key = label + "  " + styleDim.Render(desc)
		if id == activeID {
			opt = opt.Selected(true)
		}
		opts = append(opts, opt)
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Pick a session").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}

	if err := saveActiveSession(dataDir, selected); err != nil {
		return err
	}

	lipgloss.Printf("%s session %s\n", styleSuccess.Render("Activated"), selected)
	return nil
}

func newSessionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [session-id]",
		Short: "Show session details",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runSessionShowCmd,
	}
}

func runSessionShowCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		sessionID = loadActiveSession(cfg.DataDir)
		if sessionID == "" {
			return fmt.Errorf("no active session; specify a session ID or run a prompt first")
		}
	}

	svc := &sessions.FileSessionService{BaseDir: cfg.DataDir}
	info, err := svc.Get(core.SessionID(sessionID))
	if err != nil {
		return err
	}

	activeID := loadActiveSession(cfg.DataDir)
	statusText := styleDim.Render("inactive")
	if sessionID == activeID {
		statusText = styleSuccess.Render("active")
	}

	agentName := info.DefaultAgent
	if agentName == "" {
		agentName = "default"
	}

	lipgloss.Println(kvLine("Session", string(info.ID)))
	lipgloss.Println(kvLine("Status", statusText))
	lipgloss.Println(kvLine("Agent", agentName))
	lipgloss.Println(kvLine("Messages", fmt.Sprintf("%d", info.MessageCount)))
	lipgloss.Println(kvLine("Size", formatSize(info.FileSize)))
	if !info.CreatedAt.IsZero() {
		lipgloss.Println(kvLine("Created", info.CreatedAt.Format(time.RFC3339)))
	}
	lipgloss.Println(kvLine("Modified", info.ModifiedAt.Format(time.RFC3339)))

	return nil
}

func newSessionDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <session-id>",
		Short: "Delete a session",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionDeleteCmd,
	}

	cmd.Flags().Bool("force", false, "force delete the active session")

	return cmd
}

func runSessionDeleteCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	sessionID := args[0]
	activeID := loadActiveSession(cfg.DataDir)
	force, _ := cmd.Flags().GetBool("force")

	if sessionID == activeID && !force {
		return fmt.Errorf("session %s is active; use --force to delete it", sessionID)
	}

	svc := &sessions.FileSessionService{BaseDir: cfg.DataDir}
	if err := svc.Delete(core.SessionID(sessionID)); err != nil {
		return err
	}

	if sessionID == activeID {
		if err := clearActiveSession(cfg.DataDir); err != nil {
			return err
		}
	}

	lipgloss.Printf("%s session %s\n", styleSuccess.Render("Deleted"), sessionID)
	return nil
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(bytes)/1024/1024)
	case bytes >= 1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatTime(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 48*time.Hour:
		return "yesterday"
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}
