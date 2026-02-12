package main

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

	"github.com/erg0nix/kontekst/internal/acp"
	"github.com/erg0nix/kontekst/internal/config"

	"github.com/spf13/cobra"
)

func execute() {
	rootCmd := &cobra.Command{
		Use:   "kontekst [prompt]",
		Short: "kontekst CLI",
		Args:  cobra.ArbitraryArgs,
		RunE:  runCmd,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().String("server", "", "server address")
	rootCmd.PersistentFlags().Bool("auto-approve", false, "auto-approve tools")
	rootCmd.PersistentFlags().String("session", "", "session id to reuse")
	rootCmd.PersistentFlags().String("agent", "", "agent to use for this run")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newSessionCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newPsCmd())
	rootCmd.AddCommand(newLlamaCmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newACPCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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
	pidFile := filepath.Join(dataDir, "server.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func loadActiveSession(dataDir string) string {
	path := filepath.Join(dataDir, "active_session")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func clearActiveSession(dataDir string) error {
	path := filepath.Join(dataDir, "active_session")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear active session: %w", err)
	}
	return nil
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
	fmt.Println("server is not running at", addr)
	fmt.Println("start with: kontekst start")
	if err != nil {
		fmt.Println("error:", err)
	}
}

func printStatus(addr string, resp acp.StatusResponse) {
	fmt.Println("kontekst server")
	fmt.Println("  address:", addr)
	fmt.Println("  bind:", resp.Bind)
	fmt.Println("  uptime:", resp.Uptime)
	if resp.StartedAt != "" {
		fmt.Println("  started:", resp.StartedAt)
	}
	fmt.Println("  data_dir:", resp.DataDir)
}

func startServer(cfg config.Config, configPath string, foreground bool) error {
	if alreadyRunning(cfg.DataDir) {
		serverAddr := resolveServer("", cfg)
		fmt.Println("server already running at", serverAddr)
		return nil
	}

	serverCmd := exec.Command(os.Args[0], "server")
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

	fmt.Println("started server pid", serverCmd.Process.Pid)
	return nil
}
