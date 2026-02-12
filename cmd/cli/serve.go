package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/acp"
	"github.com/erg0nix/kontekst/internal/config"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start kontekst server and llama-server",
		RunE:  runServeCmd,
	}

	cmd.Flags().Bool("stdio", false, "run ACP handler over stdio (for editors)")
	cmd.Flags().Bool("foreground", false, "run server in foreground")
	cmd.Flags().String("bind", "", "bind address (overrides config)")
	cmd.Flags().String("llama-bin", "llama-server", "path to llama-server binary")

	return cmd
}

func runServeCmd(cmd *cobra.Command, _ []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	stdio, _ := cmd.Flags().GetBool("stdio")
	foreground, _ := cmd.Flags().GetBool("foreground")
	bindOverride, _ := cmd.Flags().GetString("bind")
	llamaBin, _ := cmd.Flags().GetString("llama-bin")

	cfg, _ := loadConfig(configPath)
	if bindOverride != "" {
		cfg.Bind = bindOverride
	}

	if stdio {
		return runStdio(cfg)
	}

	startLlamaServer(llamaBin)

	if foreground {
		cfg.Debug = config.LoadDebugConfigFromEnv(cfg.Debug)
		return runServer(cfg)
	}

	return startServer(cfg, configPath, false)
}

func runStdio(cfg config.Config) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	cfg.Debug = config.LoadDebugConfigFromEnv(cfg.Debug)

	services := setupServices(cfg)
	handler := acp.NewHandler(services.Runner, services.Agents, services.Skills)
	conn := handler.Serve(os.Stdout, os.Stdin)

	<-conn.Done()
	return nil
}

func runServer(cfg config.Config) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	services := setupServices(cfg)
	startTime := time.Now()

	listener, err := net.Listen("tcp", cfg.Bind)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", cfg.Bind, err)
	}

	pidFile := filepath.Join(cfg.DataDir, "server.pid")
	if err := writePIDFile(pidFile); err != nil {
		slog.Warn("failed to write PID file", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	shutdownCh := make(chan struct{})

	var wg sync.WaitGroup

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				handleConnection(conn, services, cfg, startTime, shutdownCh)
			}()
		}
	}()

	slog.Info("server listening", "address", cfg.Bind)

	select {
	case <-ctx.Done():
		slog.Info("received signal, shutting down")
	case <-shutdownCh:
		slog.Info("shutdown requested via protocol")
	}

	listener.Close()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		slog.Warn("drain timeout, forcing shutdown")
	}

	os.Remove(pidFile)
	return nil
}

func handleConnection(conn net.Conn, services setupResult, cfg config.Config, startTime time.Time, shutdownCh chan struct{}) {
	defer conn.Close()

	handler := acp.NewHandler(services.Runner, services.Agents, services.Skills)

	dispatch := func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case acp.MethodKontekstStatus:
			uptime := time.Since(startTime).Round(time.Second).String()
			return acp.StatusResponse{
				Bind:      cfg.Bind,
				Uptime:    uptime,
				StartedAt: startTime.Format(time.RFC3339),
				DataDir:   cfg.DataDir,
			}, nil

		case acp.MethodKontekstShutdown:
			go func() {
				select {
				case shutdownCh <- struct{}{}:
				default:
				}
			}()
			return map[string]any{"message": "shutting down"}, nil

		default:
			return handler.Dispatch(ctx, method, params)
		}
	}

	acpConn := handler.ServeWith(dispatch, conn, conn)
	<-acpConn.Done()
}

func writePIDFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write pid file: mkdir: %w", err)
	}

	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	return nil
}

func startLlamaServer(binPath string) {
	homeDir, _ := os.UserHomeDir()
	modelDir := filepath.Join(homeDir, "models")

	args := []string{
		"--host", "127.0.0.1",
		"--port", "8080",
		"--ctx-size", "4096",
		"--n-gpu-layers", "99",
		"--models-dir", modelDir,
		"--reasoning-format", "deepseek",
	}

	llamaCmd := exec.Command(binPath, args...)
	llamaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := llamaCmd.Start(); err != nil {
		lipgloss.Println(styleWarning.Render("llama-server not started: " + err.Error()))
		return
	}

	lipgloss.Println(
		styleSuccess.Render("started llama-server") + " " +
			stylePID.Render(fmt.Sprintf("pid %d", llamaCmd.Process.Pid)))
}
