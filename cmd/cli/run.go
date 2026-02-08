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

	var skillInvocation *pb.SkillInvocation
	if strings.HasPrefix(prompt, "/") {
		skillName, skillArgs := parseSkillInvocation(prompt)
		skillInvocation = &pb.SkillInvocation{Name: skillName, Arguments: skillArgs}
		prompt = ""
	}

	if prompt == "" && skillInvocation == nil {
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

	workingDir, _ := os.Getwd()
	startCmd := &pb.StartRunCommand{
		Prompt:     prompt,
		SessionId:  sessionID,
		AgentName:  agentName,
		WorkingDir: workingDir,
		Skill:      skillInvocation,
	}
	if err := stream.Send(&pb.RunCommand{Command: &pb.RunCommand_Start{Start: startCmd}}); err != nil {
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
		case *pb.RunEvent_TurnCompleted:
			if e.TurnCompleted.Reasoning != "" {
				fmt.Printf("[Reasoning: %s]\n\n", e.TurnCompleted.Reasoning)
			}
			if e.TurnCompleted.Content != "" {
				fmt.Println(e.TurnCompleted.Content)
			}
		case *pb.RunEvent_ContextSnapshot:
			snap := e.ContextSnapshot.Context
			if snap != nil {
				pct := int32(0)
				if snap.ContextSize > 0 {
					pct = snap.TotalTokens * 100 / snap.ContextSize
				}
				fmt.Printf("[context: %d/%d tokens (%d%%) | system:%d tools:%d history:%d memory:%d | budget remaining:%d]\n",
					snap.TotalTokens, snap.ContextSize, pct,
					snap.SystemTokens, snap.ToolTokens, snap.HistoryTokens, snap.MemoryTokens,
					snap.RemainingTokens)
			}
		case *pb.RunEvent_ToolsProposed:
			for _, c := range e.ToolsProposed.Calls {
				if autoApprove {
					_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_ApproveTool{ApproveTool: &pb.ApproveToolCommand{CallId: c.CallId}}})
					continue
				}

				fmt.Printf("tool: %s(%s)\n", c.Name, c.ArgumentsJson)
				if c.Preview != "" {
					fmt.Println("  Preview:")
					for _, line := range strings.Split(c.Preview, "\n") {
						fmt.Printf("    %s\n", line)
					}
				}

				fmt.Print("approve? [y/N]: ")
				line, _ := reader.ReadString('\n')

				if len(line) > 0 && (line[0] == 'y' || line[0] == 'Y') {
					_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_ApproveTool{ApproveTool: &pb.ApproveToolCommand{CallId: c.CallId}}})
				} else {
					_ = stream.Send(&pb.RunCommand{Command: &pb.RunCommand_DenyTool{DenyTool: &pb.DenyToolCommand{CallId: c.CallId, Reason: "user denied"}}})
				}
			}
		case *pb.RunEvent_ToolCompleted:
			fmt.Printf("tool %s completed: %s\n", e.ToolCompleted.CallId, e.ToolCompleted.Output)
		case *pb.RunEvent_ToolFailed:
			fmt.Printf("tool %s failed: %s\n", e.ToolFailed.CallId, e.ToolFailed.Error)
		case *pb.RunEvent_Completed:
			fmt.Println("\nrun completed")
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

func parseSkillInvocation(input string) (name, args string) {
	input = strings.TrimPrefix(input, "/")
	parts := strings.SplitN(input, " ", 2)
	name = parts[0]
	if len(parts) > 1 {
		args = parts[1]
	}
	return
}
