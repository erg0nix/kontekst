package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/erg0nix/kontekst/internal/acp"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/sessions"

	"github.com/spf13/cobra"
)

func runCmd(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	serverOverride, _ := cmd.Flags().GetString("server")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	sessionOverride, _ := cmd.Flags().GetString("session")
	agentName, _ := cmd.Flags().GetString("agent")

	cfg, _ := loadConfig(configPath)
	serverAddr := resolveServer(serverOverride, cfg)

	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	sessionID := strings.TrimSpace(sessionOverride)
	if sessionID == "" {
		sessionID = loadActiveSession(cfg.DataDir)
	}

	if agentName == "" && sessionID != "" {
		sessionService := &sessions.FileSessionService{BaseDir: cfg.DataDir}
		if defaultAgent, err := sessionService.GetDefaultAgent(core.SessionID(sessionID)); err == nil && defaultAgent != "" {
			agentName = defaultAgent
		}
	}
	if agentName == "" {
		agentName = agentConfig.DefaultAgentName
	}

	reader := bufio.NewReader(os.Stdin)

	client, err := dialServer(serverAddr, acp.ClientCallbacks{
		OnUpdate: func(notif acp.SessionNotification) {
			handleSessionUpdate(notif)
		},
		OnPermission: func(req acp.RequestPermissionRequest) acp.RequestPermissionResponse {
			return handlePermission(req, autoApprove, reader)
		},
		OnContextSnapshot: func(raw json.RawMessage) {
			handleContextSnapshot(raw)
		},
	})
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()

	_, err = client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersion,
		ClientInfo:      &acp.Implementation{Name: "kontekst-cli"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	var meta map[string]any
	if agentName != "" {
		meta = map[string]any{"agentName": agentName}
	}

	workingDir, _ := os.Getwd()

	var sid acp.SessionID
	if sessionID != "" {
		resp, err := client.LoadSession(ctx, acp.LoadSessionRequest{
			SessionID:  acp.SessionID(sessionID),
			Cwd:        workingDir,
			McpServers: []acp.McpServer{},
		})
		if err != nil {
			sessResp, err := client.NewSession(ctx, acp.NewSessionRequest{
				Cwd:        workingDir,
				McpServers: []acp.McpServer{},
				Meta:       meta,
			})
			if err != nil {
				return fmt.Errorf("new session: %w", err)
			}
			sid = sessResp.SessionID
		} else {
			sid = resp.SessionID
		}
	} else {
		sessResp, err := client.NewSession(ctx, acp.NewSessionRequest{
			Cwd:        workingDir,
			McpServers: []acp.McpServer{},
			Meta:       meta,
		})
		if err != nil {
			return fmt.Errorf("new session: %w", err)
		}
		sid = sessResp.SessionID
	}

	_ = saveActiveSession(cfg.DataDir, string(sid))

	promptResp, err := client.Prompt(ctx, acp.PromptRequest{
		SessionID: sid,
		Prompt:    []acp.ContentBlock{acp.TextBlock(prompt)},
	})
	if err != nil {
		fmt.Println("failed:", err)
		return nil
	}

	switch promptResp.StopReason {
	case acp.StopReasonEndTurn:
		fmt.Println("\nrun completed")
	case acp.StopReasonCancelled:
		fmt.Println("cancelled")
	default:
		fmt.Println("stopped:", promptResp.StopReason)
	}

	return nil
}

func handleSessionUpdate(notif acp.SessionNotification) {
	raw, err := json.Marshal(notif.Update)
	if err != nil {
		return
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return
	}

	updateType, _ := m["sessionUpdate"].(string)
	switch updateType {
	case "agent_message_chunk":
		if content, ok := m["content"].(map[string]any); ok {
			if text, ok := content["text"].(string); ok {
				fmt.Print(text)
			}
		}
	case "agent_thought_chunk":
		if content, ok := m["content"].(map[string]any); ok {
			if text, ok := content["text"].(string); ok {
				fmt.Printf("[Reasoning: %s]\n\n", text)
			}
		}
	case "tool_call":
		title, _ := m["title"].(string)
		rawInput := m["rawInput"]
		inputJSON, _ := json.Marshal(rawInput)
		fmt.Printf("tool: %s(%s)\n", title, string(inputJSON))
	case "tool_call_update":
		status, _ := m["status"].(string)
		toolCallID, _ := m["toolCallId"].(string)

		switch status {
		case "completed":
			if content, ok := m["content"].([]any); ok && len(content) > 0 {
				if c, ok := content[0].(map[string]any); ok {
					if inner, ok := c["content"].(map[string]any); ok {
						text, _ := inner["text"].(string)
						fmt.Printf("tool %s completed: %s\n", toolCallID, text)
					}
				}
			}
		case "failed":
			if content, ok := m["content"].([]any); ok && len(content) > 0 {
				if c, ok := content[0].(map[string]any); ok {
					if inner, ok := c["content"].(map[string]any); ok {
						text, _ := inner["text"].(string)
						fmt.Printf("tool %s failed: %s\n", toolCallID, text)
					}
				}
			}
		}
	}
}

func handlePermission(req acp.RequestPermissionRequest, autoApprove bool, reader *bufio.Reader) acp.RequestPermissionResponse {
	if autoApprove {
		return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("allow")}
	}

	title := ""
	if req.ToolCall.Title != nil {
		title = *req.ToolCall.Title
	}

	inputJSON, _ := json.Marshal(req.ToolCall.RawInput)
	fmt.Printf("tool: %s(%s)\n", title, string(inputJSON))
	fmt.Print("approve? [y/N]: ")
	line, _ := reader.ReadString('\n')

	if len(line) > 0 && (line[0] == 'y' || line[0] == 'Y') {
		return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("allow")}
	}

	return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("reject")}
}

func handleContextSnapshot(raw json.RawMessage) {
	var snap core.ContextSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return
	}

	pct := 0
	if snap.ContextSize > 0 {
		pct = snap.TotalTokens * 100 / snap.ContextSize
	}
	fmt.Printf("[context: %d/%d tokens (%d%%) | system:%d tools:%d history:%d memory:%d | budget remaining:%d]\n",
		snap.TotalTokens, snap.ContextSize, pct,
		snap.SystemTokens, snap.ToolTokens, snap.HistoryTokens, snap.MemoryTokens,
		snap.RemainingTokens)
}
