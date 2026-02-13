package main

import (
	"fmt"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/spf13/cobra"
)

func newSessionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessions",
		Short: "List sessions",
		Args:  cobra.NoArgs,
		RunE:  runSessionsCmd,
	}
}

func runSessionsCmd(cmd *cobra.Command, _ []string) error {
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
	printSessionsTable(list, activeID)
	return nil
}

func printSessionsTable(list []sessions.SessionInfo, activeID string) {
	t := newTable("", "SESSION ID", "AGENT", "MESSAGES", "SIZE", "MODIFIED")

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
