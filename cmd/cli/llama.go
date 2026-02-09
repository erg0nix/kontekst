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
		endpoint   string
		modelDir   string
		gpuLayers  int
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start llama-server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			host, port := parseEndpointHostPort(endpoint)

			args := []string{
				"--host", host,
				"--port", port,
				"--n-gpu-layers", strconv.Itoa(gpuLayers),
				"--reasoning-format", "deepseek",
			}

			if modelDir != "" {
				args = append(args, "--models-dir", modelDir)
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
	cmd.Flags().StringVar(&endpoint, "endpoint", "http://127.0.0.1:8080", "LLM endpoint URL")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "directory where models live")
	cmd.Flags().IntVar(&gpuLayers, "gpu-layers", 0, "number of GPU layers")

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
