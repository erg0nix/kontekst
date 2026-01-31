package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/core"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
	"github.com/erg0nix/kontekst/internal/sessions"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	serverOverride, _ := cmd.Flags().GetString("server")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	sessionOverride, _ := cmd.Flags().GetString("session")
	agentName, _ := cmd.Flags().GetString("agent")

	config, _ := loadConfig(configPath)
	serverAddr := resolveServer(serverOverride, config)

	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	sessionID := strings.TrimSpace(sessionOverride)
	if sessionID == "" {
		sessionID = loadActiveSession(config.DataDir)
	}

	if agentName == "" && sessionID != "" {
		sessionService := &sessions.FileSessionService{BaseDir: config.DataDir}
		if defaultAgent, err := sessionService.GetDefaultAgent(core.SessionID(sessionID)); err == nil && defaultAgent != "" {
			agentName = defaultAgent
		}
	}
	if agentName == "" {
		agentName = agentConfig.DefaultAgentName
	}

	grpcConn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		printDaemonNotRunning(serverAddr, err)
		return err
	}
	defer grpcConn.Close()

	agentClient := pb.NewAgentServiceClient(grpcConn)
	streamCtx := context.Background()
	stream, err := agentClient.Run(streamCtx)
	if err != nil {
		printDaemonNotRunning(serverAddr, err)
		return err
	}

	if err := stream.Send(&pb.RunCommand{Command: &pb.RunCommand_Start{Start: &pb.StartRunCommand{Prompt: prompt, SessionId: sessionID, AgentName: agentName}}}); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		runEvent, err := stream.Recv()
		if err != nil {
			break
		}

		switch e := runEvent.Event.(type) {
		case *pb.RunEvent_Started:
			fmt.Println("run started", e.Started.RunId)

			if e.Started.SessionId != "" {
				_ = saveActiveSession(config.DataDir, e.Started.SessionId)
			}
		case *pb.RunEvent_BatchProposed:
			if autoApprove {
				_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_ApproveAll{ApproveAll: &pb.ApproveAllToolsCommand{BatchId: e.BatchProposed.BatchId}}})
				continue
			}

			fmt.Println("tool calls proposed:")

			for _, c := range e.BatchProposed.Calls {
				fmt.Printf("- %s(%s)\n", c.Name, c.ArgumentsJson)
			}

			fmt.Print("approve all? [y/N]: ")
			line, _ := reader.ReadString('\n')

			if len(line) > 0 && (line[0] == 'y' || line[0] == 'Y') {
				_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_ApproveAll{ApproveAll: &pb.ApproveAllToolsCommand{BatchId: e.BatchProposed.BatchId}}})
			} else {
				_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_DenyAll{DenyAll: &pb.DenyAllToolsCommand{BatchId: e.BatchProposed.BatchId, Reason: "user denied"}}})
			}
		case *pb.RunEvent_ToolCompleted:
			fmt.Printf("tool %s completed: %s\n", e.ToolCompleted.CallId, e.ToolCompleted.Output)
		case *pb.RunEvent_ToolFailed:
			fmt.Printf("tool %s failed: %s\n", e.ToolFailed.CallId, e.ToolFailed.Error)
		case *pb.RunEvent_Completed:
			fmt.Println(e.Completed.Content)
			return nil
		case *pb.RunEvent_Cancelled:
			fmt.Println("cancelled")
			return nil
		case *pb.RunEvent_Failed:
			fmt.Println("failed:", e.Failed.Error)
			return nil
		}
	}
	return nil
}
