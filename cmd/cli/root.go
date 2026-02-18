package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/acp"
	"github.com/erg0nix/kontekst/internal/config"

	"github.com/spf13/cobra"
)

func execute() {
	rootCmd := &cobra.Command{
		Use:           "kontekst [prompt]",
		Short:         "kontekst CLI",
		SilenceErrors: true,
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

	if err := rootCmd.Execute(); err != nil {
		lipgloss.Println(styleError.Render(err.Error()))
		os.Exit(1)
	}
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
	return readPID(filepath.Join(dataDir, "server.pid")) != 0
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

func dialServer(serverAddr string, cb acp.ClientCallbacks) (*acp.Client, error) {
	client, err := acp.Dial(context.Background(), serverAddr, cb)
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
		return err
	}

	out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	serverCmd.Stdout = out
	serverCmd.Stderr = out

	if err := serverCmd.Start(); err != nil {
		return err
	}

	lipgloss.Println(
		styleSuccess.Render("started server") + " " +
			stylePID.Render(fmt.Sprintf("pid %d", serverCmd.Process.Pid)))
	return nil
}
