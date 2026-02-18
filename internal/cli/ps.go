package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"github.com/erg0nix/kontekst/internal/acp"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "Show running processes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}

			t := newTable("NAME", "STATUS", "PID", "ENDPOINT", "UPTIME")

			addServerRow(cmd.Context(), t, app.Config.DataDir, app.ServerAddr)
			addLlamaRow(t)

			lipgloss.Println(t.Render())
			return nil
		},
	}
}

func addServerRow(ctx context.Context, t *table.Table, dataDir string, serverAddr string) {
	pid := readPID(filepath.Join(dataDir, "server.pid"))
	if pid == 0 {
		t.Row("kontekst", styleError.Render("stopped"), "-", serverAddr, "-")
		return
	}

	var uptime string
	client, err := acp.Dial(ctx, serverAddr, acp.ClientCallbacks{})
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
	pid := findProcessPID("llama-server")
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

func readPID(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return 0
	}

	if process.Signal(syscall.Signal(0)) != nil {
		return 0
	}

	return pid
}

func findProcessPID(name string) int {
	out, err := pidofCommand(name)
	if err != nil {
		return 0
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0
	}

	pid, _ := strconv.Atoi(fields[0])
	return pid
}

func pidofCommand(name string) ([]byte, error) {
	return exec.Command("pgrep", "-f", name).Output()
}
