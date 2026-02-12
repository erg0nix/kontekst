package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

	"github.com/spf13/cobra"
)

func newLlamaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llama",
		Short: "Manage llama-server",
	}

	cmd.AddCommand(newLlamaStartCmd())
	cmd.AddCommand(newLlamaStopCmd())

	return cmd
}

func newLlamaStartCmd() *cobra.Command {
	var (
		binPath    string
		background bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start llama-server",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			if background {
				llamaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				if err := llamaCmd.Start(); err != nil {
					return fmt.Errorf("start llama-server: %w", err)
				}

				lipgloss.Println(
					styleSuccess.Render("started llama-server") + " " +
						stylePID.Render(fmt.Sprintf("pid %d", llamaCmd.Process.Pid)))
				return nil
			}

			llamaCmd.Stdout = os.Stdout
			llamaCmd.Stderr = os.Stderr

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			if err := llamaCmd.Start(); err != nil {
				return fmt.Errorf("start llama-server: %w", err)
			}

			go func() {
				<-sigCh
				if llamaCmd.Process != nil {
					_ = llamaCmd.Process.Signal(syscall.SIGTERM)
				}
			}()

			return llamaCmd.Wait()
		},
	}

	cmd.Flags().StringVar(&binPath, "bin", "llama-server", "path to llama-server binary")
	cmd.Flags().BoolVar(&background, "background", false, "run in background (detached)")

	return cmd
}

func newLlamaStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop llama-server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := exec.Command("pkill", "-f", "llama-server").CombinedOutput()
			if err != nil {
				return fmt.Errorf("stop llama-server: %s", strings.TrimSpace(string(out)))
			}

			lipgloss.Println(styleSuccess.Render("stopped llama-server"))
			return nil
		},
	}
}
