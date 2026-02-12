package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	lipgloss "github.com/charmbracelet/lipgloss/v2"

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
		if err == nil {
			sid = resp.SessionID
		}
	}

	if sid == "" {
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
		lipgloss.Println(styledError("prompt failed", err.Error()))
		return nil
	}

	switch promptResp.StopReason {
	case acp.StopReasonEndTurn:
		lipgloss.Print("\n" + styleSuccess.Render("run completed") + "\n")
	case acp.StopReasonCancelled:
		lipgloss.Println(styleWarning.Render("cancelled"))
	default:
		lipgloss.Println(styleWarning.Render("stopped: " + string(promptResp.StopReason)))
	}

	return nil
}

func handleSessionUpdate(notif acp.SessionNotification) {
	m, ok := notif.Update.(map[string]any)
	if !ok {
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
				lipgloss.Print(styleReasoning.Render(text) + "\n\n")
			}
		}
	case "tool_call":
		title, _ := m["title"].(string)
		kind, _ := m["kind"].(string)
		rawInput := m["rawInput"]
		inputJSON, _ := json.Marshal(rawInput)

		label := toolKindLabel(kind)
		labelStyled := toolKindStyle(kind).Render(label)
		nameStyled := styleToolName.Render(title)
		argsStyled := styleToolArgs.Render("(" + string(inputJSON) + ")")
		lipgloss.Println(labelStyled + " " + nameStyled + argsStyled)
	case "tool_call_update":
		status, _ := m["status"].(string)
		text := extractToolResultText(m)

		switch status {
		case "completed":
			lipgloss.Println("  " + styleSuccess.Render("done") + " " + styleDim.Render(truncate(text, 120)))
		case "failed":
			lipgloss.Println("  " + styleError.Render("fail") + " " + styleDim.Render(truncate(text, 120)))
		}
	}
}

func extractToolResultText(m map[string]any) string {
	content, ok := m["content"].([]any)
	if !ok || len(content) == 0 {
		return ""
	}

	c, ok := content[0].(map[string]any)
	if !ok {
		return ""
	}

	inner, ok := c["content"].(map[string]any)
	if !ok {
		return ""
	}

	text, _ := inner["text"].(string)
	return text
}

func truncate(s string, max int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}

	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func handlePermission(req acp.RequestPermissionRequest, autoApprove bool, reader *bufio.Reader) acp.RequestPermissionResponse {
	if autoApprove {
		return acp.RequestPermissionResponse{Outcome: acp.PermissionSelected("allow")}
	}

	title := ""
	if req.ToolCall.Title != nil {
		title = *req.ToolCall.Title
	}

	kind := ""
	if req.ToolCall.Kind != nil {
		kind = string(*req.ToolCall.Kind)
	}

	inputJSON, _ := json.Marshal(req.ToolCall.RawInput)

	label := toolKindLabel(kind)
	labelStyled := toolKindStyle(kind).Render(label)
	nameStyled := styleToolName.Render(title)
	argsStyled := styleToolArgs.Render("(" + string(inputJSON) + ")")
	lipgloss.Println(labelStyled + " " + nameStyled + argsStyled)

	lipgloss.Print(stylePromptAction.Render("approve?") + " " + stylePromptHint.Render("[y/N]") + ": ")
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

	pctStyle := styleDim
	switch {
	case pct > 95:
		pctStyle = styleError
	case pct > 80:
		pctStyle = styleWarning
	}

	pctStr := pctStyle.Render(fmt.Sprintf("%d%%", pct))
	header := styleDim.Render("ctx") + " " +
		fmt.Sprintf("%d/%d ", snap.TotalTokens, snap.ContextSize) +
		pctStr

	details := styleDim.Render(fmt.Sprintf(
		"sys:%d tools:%d hist:%d mem:%d free:%d",
		snap.SystemTokens, snap.ToolTokens, snap.HistoryTokens,
		snap.MemoryTokens, snap.RemainingTokens))

	lipgloss.Println(header)
	lipgloss.Println("  " + details)
}
