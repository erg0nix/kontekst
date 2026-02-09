package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func newLlamaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llama",
		Short: "Manage llama-server",
	}

	cmd.AddCommand(newLlamaStartCmd())

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
			configPath, _ := cmd.Flags().GetString("config")
			cfg, _ := loadConfig(configPath)

			host, port := parseEndpointHostPort(cfg.Endpoint)

			args := []string{
				"--host", host,
				"--port", port,
				"--n-gpu-layers", strconv.Itoa(cfg.GPULayers),
				"--reasoning-format", "deepseek",
			}

			if cfg.ModelDir != "" {
				args = append(args, "--models-dir", cfg.ModelDir)
			}

			llamaCmd := exec.Command(binPath, args...)

			if background {
				llamaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				if err := llamaCmd.Start(); err != nil {
					return fmt.Errorf("start llama-server: %w", err)
				}

				fmt.Println("started llama-server pid", llamaCmd.Process.Pid)
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

func parseEndpointHostPort(endpoint string) (string, string) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "127.0.0.1", "8080"
	}

	host := parsed.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}

	port := parsed.Port()
	if port == "" {
		port = "8080"
	}

	return host, port
}
