package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/app"
	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/protocol"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "kontekst [prompt]",
		Short:         "kontekst CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE:          runCmd,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().String("server", "", "server address")
	rootCmd.PersistentFlags().Bool("auto-approve", false, "auto-approve tools")
	rootCmd.PersistentFlags().String("session", "", "session id to reuse")
	rootCmd.PersistentFlags().String("agent", "", "agent to use for this run")

	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newSessionsCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newPsCmd())
	rootCmd.AddCommand(newInitCmd())

	return rootCmd
}

func loadConfig(path string) (config.Config, error) {
	configPath := path
	if configPath == "" {
		configPath = filepath.Join(config.Default().DataDir, "config.toml")
	}
	return config.LoadOrCreate(configPath)
}

func resolveServer(override string, cfg config.Config) string {
	if override != "" {
		return override
	}
	return clientAddrFromBind(cfg.Bind)
}

func clientAddrFromBind(bind string) string {
	host, port, err := netSplitHostPort(bind)
	if err != nil || port == "" {
		return bind
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		return "127.0.0.1:" + port
	}
	return bind
}

func netSplitHostPort(addr string) (string, string, error) {
	if strings.HasPrefix(addr, ":") {
		return "", strings.TrimPrefix(addr, ":"), nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", err
	}
	return host, port, nil
}

func alreadyRunning(dataDir string) bool {
	return app.ReadPID(filepath.Join(dataDir, "server.pid")) != 0
}

func loadActiveSession(dataDir string) string {
	path := filepath.Join(dataDir, "active_session")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveActiveSession(dataDir string, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("save active session: mkdir: %w", err)
	}

	path := filepath.Join(dataDir, "active_session")
	if err := os.WriteFile(path, []byte(sessionID), 0o644); err != nil {
		return fmt.Errorf("save active session: %w", err)
	}
	return nil
}

func dialServer(ctx context.Context, serverAddr string, cb protocol.ClientCallbacks) (*protocol.Client, error) {
	client, err := protocol.Dial(ctx, serverAddr, cb)
	if err != nil {
		printServerNotRunning(serverAddr, err)
		return nil, err
	}
	return client, nil
}

func printServerNotRunning(addr string, err error) {
	lipgloss.Println(styleError.Render("server is not running at " + addr))
	lipgloss.Println("start with: " + styleToolName.Render("kontekst serve"))
	if err != nil {
		lipgloss.Println(styleDim.Render(err.Error()))
	}
}

func startServer(cfg config.Config, configPath string, foreground bool) error {
	if alreadyRunning(cfg.DataDir) {
		serverAddr := resolveServer("", cfg)
		lipgloss.Println(styleDim.Render("server already running at " + serverAddr))
		return nil
	}

	serverCmd := exec.Command(os.Args[0], "serve", "--foreground")
	if configPath != "" {
		serverCmd.Args = append(serverCmd.Args, "--config", configPath)
	}

	if foreground {
		serverCmd.Stdout = os.Stdout
		serverCmd.Stderr = os.Stderr
		return serverCmd.Run()
	}

	logFile := filepath.Join(cfg.DataDir, "server.log")
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("start server: create data dir: %w", err)
	}

	out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("start server: open log: %w", err)
	}
	defer out.Close()

	serverCmd.Stdout = out
	serverCmd.Stderr = out

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	lipgloss.Println(
		styleSuccess.Render("started server") + " " +
			stylePID.Render(fmt.Sprintf("pid %d", serverCmd.Process.Pid)))
	return nil
}
