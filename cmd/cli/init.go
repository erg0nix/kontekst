package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/acp"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"

	"github.com/spf13/cobra"
)

const agentsMDFile = "AGENTS.md"

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate an AGENTS.md for the current project",
		RunE:  runInitCmd,
	}
}

func runInitCmd(cmd *cobra.Command, _ []string) error {
	if _, err := os.Stat(agentsMDFile); err == nil {
		return fmt.Errorf("%s already exists; remove it first to regenerate", agentsMDFile)
	}

	configPath, _ := cmd.Flags().GetString("config")
	serverOverride, _ := cmd.Flags().GetString("server")

	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	serverAddr := resolveServer(serverOverride, cfg)

	if !alreadyRunning(cfg.DataDir) {
		if err := startServer(cfg, configPath, false); err != nil {
			return fmt.Errorf("auto-start server: %w", err)
		}
	}

	var output strings.Builder

	allowedTools := map[string]bool{
		"read_file":  true,
		"list_files": true,
	}

	callbacks := acp.ClientCallbacks{
		OnUpdate: func(notif acp.SessionNotification) {
			m, ok := notif.Update.(map[string]any)
			if !ok {
				return
			}

			updateType, _ := m["sessionUpdate"].(string)
			switch updateType {
			case "agent_message_chunk":
				if content, ok := m["content"].(map[string]any); ok {
					if text, ok := content["text"].(string); ok {
						output.WriteString(text)
					}
				}
			case "tool_call":
				title, _ := m["title"].(string)
				rawInput := m["rawInput"]
				inputJSON, _ := json.Marshal(rawInput)
				fmt.Printf("  tool: %s(%s)\n", title, string(inputJSON))
			}
		},
		OnPermission: func(req acp.RequestPermissionRequest) acp.RequestPermissionResponse {
			toolName := ""
			if req.ToolCall.Title != nil {
				toolName = *req.ToolCall.Title
			}

			if allowedTools[toolName] {
				return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("allow")}
			}
			return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("reject")}
		},
	}

	var client *acp.Client
	for range 10 {
		client, err = acp.Dial(context.Background(), serverAddr, callbacks)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("connect to server at %s: %w", serverAddr, err)
	}
	defer client.Close()

	ctx := cmd.Context()

	_, err = client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersion,
		ClientInfo:      &acp.Implementation{Name: "kontekst-cli"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	workingDir, _ := os.Getwd()

	sessResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workingDir,
		McpServers: []acp.McpServer{},
		Meta:       map[string]any{"agentName": agentConfig.InitAgentName},
	})
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}

	fmt.Println("analyzing project...")

	promptResp, err := client.Prompt(ctx, acp.PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []acp.ContentBlock{acp.TextBlock("Analyze this project and generate an AGENTS.md file.")},
	})
	if err != nil {
		return fmt.Errorf("prompt: %w", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		return fmt.Errorf("agent stopped unexpectedly: %s", promptResp.StopReason)
	}

	content := strings.TrimSpace(output.String())
	if content == "" {
		return fmt.Errorf("agent produced no output")
	}

	if err := os.WriteFile(agentsMDFile, []byte(content+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", agentsMDFile, err)
	}

	fmt.Printf("wrote %s (%d bytes)\n", agentsMDFile, len(content)+1)
	return nil
}
