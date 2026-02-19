package cli

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"github.com/erg0nix/kontekst/internal/app"
	"github.com/erg0nix/kontekst/internal/protocol"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "Show running processes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newApp(cmd)
			if err != nil {
				return err
			}

			t := newTable("NAME", "STATUS", "PID", "ENDPOINT", "UPTIME")

			addServerRow(cmd.Context(), t, a.Config.DataDir, a.ServerAddr)
			addLlamaRow(t)

			lipgloss.Println(t.Render())
			return nil
		},
	}
}

func addServerRow(ctx context.Context, t *table.Table, dataDir string, serverAddr string) {
	pid := app.ReadPID(filepath.Join(dataDir, "server.pid"))
	if pid == 0 {
		t.Row("kontekst", styleError.Render("stopped"), "-", serverAddr, "-")
		return
	}

	var uptime string
	client, err := protocol.Dial(ctx, serverAddr, protocol.ClientCallbacks{})
	if err == nil {
		defer client.Close()
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if resp, err := client.Status(ctx); err == nil {
			uptime = resp.Uptime
		}
	}

	if uptime == "" {
		uptime = "-"
	}

	t.Row("kontekst",
		styleSuccess.Render("running"),
		fmt.Sprintf("%d", pid),
		serverAddr,
		uptime)
}

func addLlamaRow(t *table.Table) {
	pid := app.FindProcessPID("llama-server")
	if pid == 0 {
		t.Row("llama-server", styleError.Render("stopped"), "-", "127.0.0.1:8080", "-")
		return
	}

	endpoint := "127.0.0.1:8080"
	status := styleSuccess.Render("running")

	conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
	if err != nil {
		status = styleWarning.Render("starting")
	} else {
		conn.Close()
	}

	t.Row("llama-server", status, fmt.Sprintf("%d", pid), endpoint, "-")
}
