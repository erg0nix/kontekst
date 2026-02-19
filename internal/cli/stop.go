package cli

import (
	"context"
	"os/exec"
	"strings"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/protocol"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop kontekst server and llama-server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}

			stopKontekstServer(cmd.Context(), app.ServerAddr)
			stopLlamaServer()
			return nil
		},
	}
}

func stopKontekstServer(ctx context.Context, serverAddr string) {
	client, err := protocol.Dial(ctx, serverAddr, protocol.ClientCallbacks{})
	if err != nil {
		lipgloss.Println(styleDim.Render("kontekst server not running"))
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Shutdown(ctx); err != nil {
		lipgloss.Println(styleError.Render("kontekst server: " + err.Error()))
		return
	}

	lipgloss.Println(styleSuccess.Render("stopped kontekst server"))
}

func stopLlamaServer() {
	out, err := exec.Command("pkill", "-f", "llama-server").CombinedOutput()
	if err != nil {
		lipgloss.Println(styleDim.Render("llama-server not running") + " " +
			styleDim.Render(strings.TrimSpace(string(out))))
		return
	}

	lipgloss.Println(styleSuccess.Render("stopped llama-server"))
}
