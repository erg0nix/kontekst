package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
	"github.com/erg0nix/kontekst/internal/protocol"

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

	app, err := newApp(cmd)
	if err != nil {
		return err
	}

	if !alreadyRunning(app.Config.DataDir) {
		if err := startServer(app.Config, app.ConfigPath, false); err != nil {
			return fmt.Errorf("auto-start server: %w", err)
		}
	}

	var output strings.Builder

	allowedTools := map[string]bool{
		"read_file":  true,
		"list_files": true,
	}

	callbacks := protocol.ClientCallbacks{
		OnUpdate: func(notif protocol.SessionNotification) {
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
		OnPermission: func(req protocol.RequestPermissionRequest) protocol.RequestPermissionResponse {
			toolName := ""
			if req.ToolCall.Title != nil {
				toolName = *req.ToolCall.Title
			}

			if allowedTools[toolName] {
				return protocol.RequestPermissionResponse{Outcome: protocol.PermissionSelected("allow")}
			}
			return protocol.RequestPermissionResponse{Outcome: protocol.PermissionSelected("reject")}
		},
	}

	var client *protocol.Client
	for range 10 {
		client, err = protocol.Dial(context.Background(), app.ServerAddr, callbacks)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("connect to server at %s: %w", app.ServerAddr, err)
	}
	defer client.Close()

	ctx := cmd.Context()

	_, err = client.Initialize(ctx, protocol.InitializeRequest{
		ProtocolVersion: protocol.ProtocolVersion,
		ClientInfo:      &protocol.Implementation{Name: "kontekst-cli"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	workingDir, _ := os.Getwd()

	sessResp, err := client.NewSession(ctx, protocol.NewSessionRequest{
		Cwd:        workingDir,
		McpServers: []protocol.McpServer{},
		Meta:       map[string]any{"agentName": agentConfig.InitAgentName},
	})
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}

	fmt.Println("analyzing project...")

	promptResp, err := client.Prompt(ctx, protocol.PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []protocol.ContentBlock{protocol.TextBlock("Analyze this project and generate an AGENTS.md file.")},
	})
	if err != nil {
		return fmt.Errorf("prompt: %w", err)
	}

	if promptResp.StopReason != protocol.StopReasonEndTurn {
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
