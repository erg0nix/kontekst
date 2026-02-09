package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/erg0nix/kontekst/internal/grpc/pb"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the kontekst daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			serverOverride, _ := cmd.Flags().GetString("server")
			config, _ := loadConfig(configPath)
			serverAddr := resolveServer(serverOverride, config)

			grpcConn, err := dialDaemon(serverAddr)
			if err != nil {
				return err
			}
			defer grpcConn.Close()

			daemonClient := pb.NewDaemonServiceClient(grpcConn)
			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelShutdown()

			shutdownResp, err := daemonClient.Shutdown(shutdownCtx, &pb.ShutdownRequest{})
			if err != nil {
				printDaemonNotRunning(serverAddr, err)
				return err
			}

			fmt.Println(shutdownResp.Message)
			return nil
		},
	}

	return cmd
}
