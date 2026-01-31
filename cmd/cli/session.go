package main

import (
	"fmt"

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
