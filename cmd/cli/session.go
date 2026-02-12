package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/spf13/cobra"
)

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session management commands",
	}

	cmd.AddCommand(newSessionSetAgentCmd())
	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionShowCmd())
	cmd.AddCommand(newSessionDeleteCmd())

	return cmd
}

func newSessionSetAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-agent <agent-name>",
		Short: "Set the default agent for the current session",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionSetAgentCmd,
	}

	return cmd
}

func runSessionSetAgentCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	sessionID := loadActiveSession(cfg.DataDir)
	if sessionID == "" {
		return fmt.Errorf("no active session; run a prompt first to create a session")
	}

	agentName := args[0]
	sessionService := &sessions.FileSessionService{BaseDir: cfg.DataDir}

	if err := sessionService.SetDefaultAgent(core.SessionID(sessionID), agentName); err != nil {
		return fmt.Errorf("failed to set default agent: %w", err)
	}

	fmt.Printf("Set default agent for session %s to %q\n", sessionID, agentName)
	return nil
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
		fmt.Println("No sessions found.")
		return nil
	}

	activeID := loadActiveSession(cfg.DataDir)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "   SESSION ID\tAGENT\tMESSAGES\tSIZE\tMODIFIED")

	for _, info := range list {
		marker := "  "
		if string(info.ID) == activeID {
			marker = "* "
		}

		agent := info.DefaultAgent
		if agent == "" {
			agent = "default"
		}

		fmt.Fprintf(w, "%s%s\t%s\t%d\t%s\t%s\n",
			marker, string(info.ID), agent, info.MessageCount,
			formatSize(info.FileSize), formatTime(info.ModifiedAt))
	}

	w.Flush()
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
	status := "inactive"
	if sessionID == activeID {
		status = "active"
	}

	agent := info.DefaultAgent
	if agent == "" {
		agent = "default"
	}

	fmt.Printf("Session:  %s\n", info.ID)
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("Agent:    %s\n", agent)
	fmt.Printf("Messages: %d\n", info.MessageCount)
	fmt.Printf("Size:     %s\n", formatSize(info.FileSize))
	if !info.CreatedAt.IsZero() {
		fmt.Printf("Created:  %s\n", info.CreatedAt.Format(time.RFC3339))
	}
	fmt.Printf("Modified: %s\n", info.ModifiedAt.Format(time.RFC3339))

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

	fmt.Printf("Deleted session %s\n", sessionID)
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
