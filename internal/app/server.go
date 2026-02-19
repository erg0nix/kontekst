package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/config"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
	"github.com/erg0nix/kontekst/internal/protocol"
	"github.com/erg0nix/kontekst/internal/protocol/types"
)

// RunServer starts the TCP server, listens for connections, and shuts down on signal or request.
func RunServer(cfg config.Config) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	services := NewServices(cfg)
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

func handleConnection(conn net.Conn, services Services, cfg config.Config, startTime time.Time, shutdownCh chan struct{}) {
	defer conn.Close()

	handler := protocol.NewHandler(services.Runner, services.Agents, services.Skills)

	dispatch := func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case types.MethodKontekstStatus:
			uptime := time.Since(startTime).Round(time.Second).String()
			return types.StatusResponse{
				Bind:      cfg.Bind,
				Uptime:    uptime,
				StartedAt: startTime.Format(time.RFC3339),
				DataDir:   cfg.DataDir,
			}, nil

		case types.MethodKontekstShutdown:
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

// MaxAgentContextSize returns the largest context_size across all configured agents.
func MaxAgentContextSize(dataDir string) int {
	if err := agentConfig.EnsureDefaults(dataDir); err != nil {
		slog.Warn("failed to ensure default agents", "error", err)
	}

	registry := agent.NewRegistry(dataDir)
	agents, err := registry.List()
	if err != nil {
		return 0
	}

	maxSize := 0
	for _, a := range agents {
		cfg, err := registry.Load(a.Name)
		if err != nil {
			continue
		}
		if cfg.ContextSize > maxSize {
			maxSize = cfg.ContextSize
		}
	}
	return maxSize
}
