package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func execute() {
	rootCmd := &cobra.Command{
		Use:   "kontekst [prompt]",
		Short: "kontekst CLI",
		Args:  cobra.ArbitraryArgs,
		RunE:  runCmd,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().String("server", "", "gRPC server address")
	rootCmd.PersistentFlags().Bool("auto-approve", false, "auto-approve tools")
	rootCmd.PersistentFlags().String("session", "", "session id to reuse")
	rootCmd.PersistentFlags().String("agent", "", "agent to use for this run")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newSessionCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newPsCmd())

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

func alreadyRunning(addr string) bool {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false
	}
	defer conn.Close()

	client := pb.NewDaemonServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.GetStatus(ctx, &pb.GetStatusRequest{})

	return err == nil
}

func defaultDaemonBin() string {
	executablePath, err := os.Executable()
	if err != nil {
		return "kontekst-daemon"
	}

	executableDir := filepath.Dir(executablePath)
	daemonPath := filepath.Join(executableDir, "kontekst-daemon")
	if _, err := os.Stat(daemonPath); err == nil {
		return daemonPath
	}

	return "kontekst-daemon"
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
		return err
	}

	path := filepath.Join(dataDir, "active_session")

	return os.WriteFile(path, []byte(sessionID), 0o644)
}

func dialDaemon(serverAddr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		printDaemonNotRunning(serverAddr, err)
		return nil, fmt.Errorf("dial daemon at %s: %w", serverAddr, err)
	}

	return conn, nil
}

func printDaemonNotRunning(addr string, err error) {
	fmt.Println("daemon is not running at", addr)
	fmt.Println("start with: kontekst start")
	if err != nil {
		fmt.Println("error:", err)
	}
}

func printStatus(addr string, resp *pb.GetStatusResponse) {
	fmt.Println("kontekst daemon")
	fmt.Println("  address:", addr)
	fmt.Println("  bind:", resp.Bind)
	fmt.Println("  uptime:", formatUptime(resp.UptimeSeconds))

	if resp.StartedAtRfc3339 != "" {
		fmt.Println("  started:", resp.StartedAtRfc3339)
	}

	fmt.Println("  data_dir:", resp.DataDir)
	fmt.Println("  model_dir:", resp.ModelDir)
	fmt.Println("  endpoint:", resp.Endpoint)
	fmt.Println("llama-server")
	fmt.Println("  running:", resp.LlamaServerRunning)
	fmt.Println("  healthy:", resp.LlamaServerHealthy)

	if resp.LlamaServerPid != 0 {
		fmt.Println("  pid:", resp.LlamaServerPid)
	}
}

func formatUptime(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	uptime := time.Duration(seconds) * time.Second
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	secondsRemainder := int(uptime.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, secondsRemainder)
	}

	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, secondsRemainder)
	}

	return fmt.Sprintf("%ds", secondsRemainder)
}

func startDaemon(cfg config.Config, configPath string, daemonBin string, foreground bool) error {
	serverAddr := resolveServer("", cfg)

	if alreadyRunning(serverAddr) {
		fmt.Println("daemon already running at", serverAddr)
		return nil
	}

	daemonPath := daemonBin
	if daemonPath == "" {
		daemonPath = defaultDaemonBin()
	}

	daemonCmd := exec.Command(daemonPath)
	if configPath != "" {
		daemonCmd.Args = append(daemonCmd.Args, "--config", configPath)
	}

	if foreground {
		daemonCmd.Stdout = os.Stdout
		daemonCmd.Stderr = os.Stderr
		return daemonCmd.Run()
	}

	logFile := filepath.Join(cfg.DataDir, "daemon.log")

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	daemonCmd.Stdout = out
	daemonCmd.Stderr = out

	if err := daemonCmd.Start(); err != nil {
		return err
	}

	fmt.Println("started daemon pid", daemonCmd.Process.Pid)

	return nil
}
