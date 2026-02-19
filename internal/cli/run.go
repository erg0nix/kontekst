package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"

	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
	"github.com/erg0nix/kontekst/internal/conversation"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/protocol"
	"github.com/erg0nix/kontekst/internal/session"

	"github.com/spf13/cobra"
)

func runCmd(cmd *cobra.Command, args []string) error {
	app, err := newApp(cmd)
	if err != nil {
		return err
	}
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	sessionOverride, _ := cmd.Flags().GetString("session")
	agentName, _ := cmd.Flags().GetString("agent")

	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	sessionID := strings.TrimSpace(sessionOverride)
	if sessionID == "" {
		sessionID = loadActiveSession(app.Config.DataDir)
	}

	if agentName == "" && sessionID != "" {
		sessionService := &session.FileService{BaseDir: app.Config.DataDir}
		if defaultAgent, err := sessionService.GetDefaultAgent(core.SessionID(sessionID)); err == nil && defaultAgent != "" {
			agentName = defaultAgent
		}
	}
	if agentName == "" {
		agentName = agentConfig.DefaultAgentName
	}

	ctx := cmd.Context()
	reader := bufio.NewReader(os.Stdin)
	renderer := newMarkdownRenderer()
	var lastSnapshot *conversation.Snapshot

	client, err := dialServer(ctx, app.ServerAddr, protocol.ClientCallbacks{
		OnUpdate: func(notif protocol.SessionNotification) {
			handleSessionUpdate(notif, renderer)
		},
		OnPermission: func(req protocol.RequestPermissionRequest) protocol.RequestPermissionResponse {
			return handlePermission(req, autoApprove, reader)
		},
		OnContextSnapshot: func(raw json.RawMessage) {
			var snap conversation.Snapshot
			if err := json.Unmarshal(raw, &snap); err == nil {
				lastSnapshot = &snap
			}
		},
	})
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Initialize(ctx, protocol.InitializeRequest{
		ProtocolVersion: protocol.ProtocolVersion,
		ClientInfo:      &protocol.Implementation{Name: "kontekst-cli"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	var meta map[string]any
	if agentName != "" {
		meta = map[string]any{"agentName": agentName}
	}

	workingDir, _ := os.Getwd()

	var sid protocol.SessionID
	if sessionID != "" {
		resp, err := client.LoadSession(ctx, protocol.LoadSessionRequest{
			SessionID:  protocol.SessionID(sessionID),
			Cwd:        workingDir,
			McpServers: []protocol.McpServer{},
		})
		if err == nil {
			sid = resp.SessionID
		}
	}

	if sid == "" {
		sessResp, err := client.NewSession(ctx, protocol.NewSessionRequest{
			Cwd:        workingDir,
			McpServers: []protocol.McpServer{},
			Meta:       meta,
		})
		if err != nil {
			return fmt.Errorf("new session: %w", err)
		}
		sid = sessResp.SessionID
	}

	if err := saveActiveSession(app.Config.DataDir, string(sid)); err != nil {
		slog.Warn("failed to save active session", "error", err)
	}

	promptResp, err := client.Prompt(ctx, protocol.PromptRequest{
		SessionID: sid,
		Prompt:    []protocol.ContentBlock{protocol.TextBlock(prompt)},
	})
	if err != nil {
		lipgloss.Println(styledError("prompt failed", err.Error()))
		return nil
	}

	if lastSnapshot != nil {
		fmt.Println()
		printContextSnapshot(*lastSnapshot)
	}

	switch promptResp.StopReason {
	case protocol.StopReasonEndTurn:
		lipgloss.Print("\n" + styleSuccess.Render("run completed") + "\n")
	case protocol.StopReasonCancelled:
		lipgloss.Println(styleWarning.Render("cancelled"))
	default:
		lipgloss.Println(styleWarning.Render("stopped: " + string(promptResp.StopReason)))
	}

	return nil
}

func handleSessionUpdate(notif protocol.SessionNotification, renderer *glamour.TermRenderer) {
	m, ok := notif.Update.(map[string]any)
	if !ok {
		return
	}

	updateType, _ := m["sessionUpdate"].(string)
	switch updateType {
	case "agent_message_chunk":
		if content, ok := m["content"].(map[string]any); ok {
			if text, ok := content["text"].(string); ok {
				if renderer != nil {
					if rendered, err := renderer.Render(text); err == nil {
						fmt.Print(rendered)
						return
					}
				}
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

func handlePermission(req protocol.RequestPermissionRequest, autoApprove bool, reader *bufio.Reader) protocol.RequestPermissionResponse {
	if autoApprove {
		return protocol.RequestPermissionResponse{Outcome: protocol.PermissionSelected("allow")}
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
		return protocol.RequestPermissionResponse{Outcome: protocol.PermissionSelected("allow")}
	}

	return protocol.RequestPermissionResponse{Outcome: protocol.PermissionSelected("reject")}
}

func compactStyle() ansi.StyleConfig {
	var style ansi.StyleConfig
	if termenv.HasDarkBackground() {
		style = glamourstyles.DarkStyleConfig
	} else {
		style = glamourstyles.LightStyleConfig
	}

	zero := uint(0)
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	return style
}

func newMarkdownRenderer() *glamour.TermRenderer {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width = 80
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(compactStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	return r
}

func printContextSnapshot(snap conversation.Snapshot) {
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
	line := styleDim.Render("ctx") + " " +
		fmt.Sprintf("%d/%d ", snap.TotalTokens, snap.ContextSize) +
		pctStr + "  " +
		styleDim.Render(fmt.Sprintf(
			"sys:%d tools:%d hist:%d mem:%d free:%d",
			snap.SystemTokens, snap.ToolTokens, snap.HistoryTokens,
			snap.MemoryTokens, snap.RemainingTokens))

	lipgloss.Println(line)
}
