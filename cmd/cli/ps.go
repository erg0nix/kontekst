package main

import (
	"context"
	"time"

	pb "github.com/erg0nix/kontekst/internal/grpc/pb"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			serverOverride, _ := cmd.Flags().GetString("server")
			config, _ := loadConfig(configPath)
			serverAddr := resolveServer(serverOverride, config)

			grpcConn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				printDaemonNotRunning(serverAddr, err)
				return err
			}
			defer grpcConn.Close()

			daemonClient := pb.NewDaemonServiceClient(grpcConn)
			statusCtx, cancelStatus := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelStatus()

			statusResp, err := daemonClient.GetStatus(statusCtx, &pb.GetStatusRequest{})
			if err != nil {
				printDaemonNotRunning(serverAddr, err)
				return err
			}

			printStatus(serverAddr, statusResp)
			return nil
		},
	}

	return cmd
}
