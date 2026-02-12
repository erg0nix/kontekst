package main

import (
	"context"
	"net"
	"net/url"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/acp"
	"github.com/erg0nix/kontekst/internal/agent"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "Show server status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			serverOverride, _ := cmd.Flags().GetString("server")
			cfg, _ := loadConfig(configPath)
			serverAddr := resolveServer(serverOverride, cfg)

			client, err := dialServer(serverAddr, acp.ClientCallbacks{})
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			statusResp, err := client.Status(ctx)
			if err != nil {
				printServerNotRunning(serverAddr, err)
				return err
			}

			printStatus(serverAddr, statusResp)
			printAgentProviders(cfg.DataDir)
			return nil
		},
	}

	return cmd
}

func printAgentProviders(dataDir string) {
	registry := agent.NewRegistry(dataDir)
	agentList, err := registry.List()
	if err != nil || len(agentList) == 0 {
		return
	}

	lipgloss.Print("\n")
	lipgloss.Println(styleServerName.Render("agents"))

	for _, summary := range agentList {
		if !summary.HasConfig {
			continue
		}

		cfg, err := registry.Load(summary.Name)
		if err != nil || cfg.Provider.Endpoint == "" {
			continue
		}

		lipgloss.Println(kvLine("agent", summary.Name))
		lipgloss.Println(kvLine("  endpoint", cfg.Provider.Endpoint))
		lipgloss.Println(kvLine("  model", cfg.Provider.Model))
		lipgloss.Println(kvLine("  status", checkProviderConnection(cfg.Provider.Endpoint)))
	}
}

func checkProviderConnection(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return styleError.Render("unreachable")
	}

	host := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "https":
			host = host + ":443"
		default:
			host = host + ":80"
		}
	}

	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return styleError.Render("unreachable")
	}
	conn.Close()

	return styleSuccess.Render("reachable")
}
