package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/erg0nix/kontekst/internal/app"
	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/protocol"

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
	a, err := newApp(cmd)
	if err != nil {
		return err
	}
	stdio, _ := cmd.Flags().GetBool("stdio")
	foreground, _ := cmd.Flags().GetBool("foreground")
	bindOverride, _ := cmd.Flags().GetString("bind")
	llamaBin, _ := cmd.Flags().GetString("llama-bin")

	cfg := a.Config
	if bindOverride != "" {
		cfg.Bind = bindOverride
	}

	if stdio {
		return runStdio(cfg)
	}

	startLlamaServer(llamaBin)

	if foreground {
		cfg.Debug = config.LoadDebugConfigFromEnv(cfg.Debug)
		return app.RunServer(cfg)
	}

	return startServer(cfg, a.ConfigPath, false)
}

func runStdio(cfg config.Config) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	cfg.Debug = config.LoadDebugConfigFromEnv(cfg.Debug)

	services := app.NewServices(cfg)
	handler := protocol.NewHandler(services.Runner, services.Agents, services.Skills)
	conn := handler.Serve(os.Stdout, os.Stdin)

	<-conn.Done()
	return nil
}

func startLlamaServer(binPath string) {
	homeDir, _ := os.UserHomeDir()
	modelDir := filepath.Join(homeDir, "models")
	dataDir := config.Default().DataDir

	ctxSize := app.MaxAgentContextSize(dataDir)
	if ctxSize == 0 {
		ctxSize = config.FallbackContextSize
	}

	args := []string{
		"--host", "127.0.0.1",
		"--port", "8080",
		"--ctx-size", strconv.Itoa(ctxSize),
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
