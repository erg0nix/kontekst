package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/erg0nix/kontekst/internal/agent"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/protocol/types"
	"github.com/erg0nix/kontekst/internal/skill"
)

// Handler is the server-side ACP request handler that manages sessions and routes agent events.
type Handler struct {
	runner   agent.Runner
	registry *agent.Registry
	skills   *skill.Registry
	conn     *Connection
	sessions sync.Map
	caps     types.ClientCapabilities
}

// NewHandler creates a Handler with the given agent runner, registry, and skills registry.
func NewHandler(runner agent.Runner, registry *agent.Registry, skillsRegistry *skill.Registry) *Handler {
	return &Handler{
		runner:   runner,
		registry: registry,
		skills:   skillsRegistry,
	}
}

// Serve creates a Connection using the handler's default dispatch and returns it.
func (h *Handler) Serve(w io.Writer, r io.Reader) *Connection {
	conn := NewConnection(h.Dispatch, w, r)
	h.conn = conn
	return conn
}

// ServeWith creates a Connection using a custom dispatch function and returns it.
func (h *Handler) ServeWith(dispatch MethodHandler, w io.Writer, r io.Reader) *Connection {
	conn := NewConnection(dispatch, w, r)
	h.conn = conn
	return conn
}

// Dispatch routes an incoming JSON-RPC method call to the appropriate handler.
//
// ACP: Client → Server methods:
//
//	"initialize"          → [handleInitialize]  — protocol handshake
//	"authenticate"        → no-op               — reserved for future auth
//	"session/new"         → [handleNewSession]   — create session
//	"session/load"        → [handleLoadSession]  — resume session
//	"session/prompt"      → [handlePrompt]       — run agent loop (long-lived)
//	"session/cancel"      → [handleCancel]       — cancel active prompt (notification)
//	"session/set_mode"    → no-op stub
//	"session/set_config"  → no-op stub
func (h *Handler) Dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case types.MethodInitialize:
		return h.handleInitialize(ctx, params)
	case types.MethodAuthenticate:
		return types.AuthenticateResponse{}, nil
	case types.MethodSessionNew:
		return h.handleNewSession(ctx, params)
	case types.MethodSessionLoad:
		return h.handleLoadSession(ctx, params)
	case types.MethodSessionPrompt:
		return h.handlePrompt(ctx, params)
	case types.MethodSessionCancel:
		h.handleCancel(params)
		return nil, nil
	case types.MethodSessionSetMode:
		return types.SetSessionModeResponse{}, nil
	case types.MethodSessionSetConfig:
		return types.SetSessionConfigOptionResponse{ConfigOptions: []types.SessionConfigOption{}}, nil
	default:
		if strings.HasPrefix(method, "_") {
			return nil, NewRPCError(types.ErrMethodNotFound, fmt.Sprintf("unknown extension: %s", method))
		}
		return nil, NewRPCError(types.ErrMethodNotFound, fmt.Sprintf("unknown method: %s", method))
	}
}

// handleInitialize performs the ACP handshake.
//
// ACP: "initialize"
// Request:  [types.InitializeRequest] — protocol version and client capabilities.
// Response: [types.InitializeResponse] — server capabilities, agent info, and auth methods.
//
// Stores the client's capabilities for later use (e.g. determining ACP tool support).
func (h *Handler) handleInitialize(_ context.Context, params json.RawMessage) (types.InitializeResponse, error) {
	var req types.InitializeRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &req); err != nil {
			return types.InitializeResponse{}, NewRPCError(types.ErrInvalidParams, err.Error())
		}
	}

	h.caps = req.ClientCapabilities

	return types.InitializeResponse{
		ProtocolVersion: types.ProtocolVersion,
		AgentCapabilities: types.AgentCapabilities{
			LoadSession: true,
		},
		AgentInfo: &types.Implementation{
			Name:    "kontekst",
			Title:   "Kontekst",
			Version: "0.1.0",
		},
		AuthMethods: []types.AuthMethod{},
	}, nil
}

// handleNewSession creates a fresh session with an agent.
//
// ACP: "session/new"
// Request:  [types.NewSessionRequest] — working directory, MCP servers, and optional agent name in _meta.
// Response: [types.NewSessionResponse] — the new session ID.
//
// Also pushes an available_commands_update notification if skills are registered.
func (h *Handler) handleNewSession(ctx context.Context, params json.RawMessage) (types.NewSessionResponse, error) {
	var req types.NewSessionRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return types.NewSessionResponse{}, NewRPCError(types.ErrInvalidParams, err.Error())
	}

	agentName := agentConfig.DefaultAgentName
	if req.Meta != nil {
		if name, ok := req.Meta["agentName"].(string); ok && name != "" {
			agentName = name
		}
	}

	sid := types.SessionID(core.NewSessionID())

	h.sessions.Store(sid, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(sid),
		cwd:       req.Cwd,
	})

	if h.skills != nil {
		var commands []types.Command
		for _, s := range h.skills.ModelInvocableSkills() {
			commands = append(commands, types.Command{Name: s.Name, Description: s.Description})
		}
		if len(commands) > 0 {
			_ = h.conn.Notify(ctx, types.MethodSessionUpdate, types.SessionNotification{
				SessionID: sid,
				Update:    types.AvailableCommandsUpdate(commands),
			})
		}
	}

	return types.NewSessionResponse{SessionID: sid}, nil
}

// handleLoadSession resumes an existing session by ID.
//
// ACP: "session/load"
// Request:  [types.LoadSessionRequest] — session ID and working directory.
// Response: [types.LoadSessionResponse] — the confirmed session ID.
func (h *Handler) handleLoadSession(_ context.Context, params json.RawMessage) (types.LoadSessionResponse, error) {
	var req types.LoadSessionRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return types.LoadSessionResponse{}, NewRPCError(types.ErrInvalidParams, err.Error())
	}

	agentName := agentConfig.DefaultAgentName

	h.sessions.Store(req.SessionID, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(req.SessionID),
		cwd:       req.Cwd,
	})

	return types.LoadSessionResponse{SessionID: req.SessionID}, nil
}

// handlePrompt runs the agent loop for a user prompt.
//
// ACP: "session/prompt"
// Request:  [types.PromptRequest] — session ID and prompt content blocks.
// Response: [types.PromptResponse] — stop reason (end_turn, cancelled, etc.).
//
// This is a long-lived request. While the agent runs, the server streams
// session/update notifications (text chunks, tool calls, tool results) and
// may send session/request_permission requests back to the client.
// The response is returned only when the agent loop completes.
func (h *Handler) handlePrompt(ctx context.Context, params json.RawMessage) (types.PromptResponse, error) {
	var req types.PromptRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return types.PromptResponse{}, NewRPCError(types.ErrInvalidParams, err.Error())
	}

	val, ok := h.sessions.Load(req.SessionID)
	if !ok {
		return types.PromptResponse{}, NewRPCError(types.ErrNotFound, "session not found")
	}
	sess := val.(*sessionState)

	promptText := extractText(req.Prompt)

	var skill *skill.Skill
	var skillContent string
	if strings.HasPrefix(promptText, "/") {
		skillName, args := parseSkillInvocation(promptText)
		if h.skills != nil {
			if loadedSkill, found := h.skills.Get(skillName); found {
				rendered, err := loadedSkill.Render(args)
				if err != nil {
					return types.PromptResponse{}, NewRPCError(types.ErrInvalidParams, fmt.Sprintf("failed to render skill: %v", err))
				}
				skill = loadedSkill
				skillContent = rendered
				promptText = args
			}
		}
	}

	agentCfg, err := h.registry.Load(sess.agentName)
	if err != nil {
		return types.PromptResponse{}, NewRPCError(types.ErrInvalidParams, err.Error())
	}

	runCtx, cancelFn := context.WithCancel(ctx)

	runCfg := agent.RunConfig{
		Prompt:              promptText,
		SessionID:           sess.sessionID,
		AgentName:           sess.agentName,
		AgentSystemPrompt:   agentCfg.SystemPrompt,
		ContextSize:         agentCfg.ContextSize,
		Sampling:            agentCfg.Sampling,
		ProviderEndpoint:    agentCfg.Provider.Endpoint,
		ProviderModel:       agentCfg.Provider.Model,
		ProviderHTTPTimeout: agentCfg.Provider.HTTPTimeout,
		WorkingDir:          sess.cwd,
		Skill:               skill,
		SkillContent:        skillContent,
		ToolRole:            agentCfg.ToolRole,
	}

	if hasACPTools(h.caps) {
		runCfg.Tools = NewToolExecutor(h.conn, req.SessionID, h.caps)
	}

	commandCh, eventCh, err := h.runner.StartRun(runCfg)
	if err != nil {
		cancelFn()
		return types.PromptResponse{}, NewRPCError(types.ErrInternalError, err.Error())
	}

	sess.mu.Lock()
	sess.cancelFn = cancelFn
	sess.commandCh = commandCh
	sess.doneCh = make(chan struct{})
	sess.mu.Unlock()

	return h.forwardEvents(runCtx, req.SessionID, sess, eventCh)
}

func (h *Handler) forwardEvents(ctx context.Context, sid types.SessionID, sess *sessionState, eventCh <-chan agent.Event) (types.PromptResponse, error) {
	defer func() {
		sess.mu.Lock()
		if sess.doneCh != nil {
			close(sess.doneCh)
		}
		if sess.cancelFn != nil {
			sess.cancelFn()
		}
		sess.commandCh = nil
		sess.cancelFn = nil
		sess.doneCh = nil
		sess.mu.Unlock()
	}()

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return types.PromptResponse{StopReason: types.StopReasonEndTurn}, nil
			}

			resp, done, err := h.processEvent(ctx, sid, sess, event)
			if err != nil {
				return types.PromptResponse{}, err
			}
			if done {
				return resp, nil
			}

		case <-h.conn.Done():
			sess.sendCommand(agent.Command{Type: agent.CmdCancel})
			return types.PromptResponse{StopReason: types.StopReasonCancelled}, nil

		case <-ctx.Done():
			sess.sendCommand(agent.Command{Type: agent.CmdCancel})
			return types.PromptResponse{StopReason: types.StopReasonCancelled}, nil
		}
	}
}

func (h *Handler) processEvent(ctx context.Context, sid types.SessionID, sess *sessionState, event agent.Event) (types.PromptResponse, bool, error) {
	switch event.Type {
	case agent.EvtRunStarted:
		return types.PromptResponse{}, false, nil

	case agent.EvtTokenDelta:
		h.sendUpdate(ctx, sid, types.AgentMessageChunk(event.Token))
		return types.PromptResponse{}, false, nil

	case agent.EvtReasoningDelta:
		h.sendUpdate(ctx, sid, types.AgentThoughtChunk(event.Reasoning))
		return types.PromptResponse{}, false, nil

	case agent.EvtTurnCompleted:
		if event.Response.Reasoning != "" {
			h.sendUpdate(ctx, sid, types.AgentThoughtChunk(event.Response.Reasoning))
		}
		if event.Response.Content != "" {
			h.sendUpdate(ctx, sid, types.AgentMessageChunk(event.Response.Content))
		}
		if event.Snapshot != nil {
			_ = h.conn.Notify(ctx, types.MethodKontekstContext, event.Snapshot)
		}
		return types.PromptResponse{}, false, nil

	case agent.EvtToolsProposed:
		for _, call := range event.Calls {
			rawInput := parseRawInput(call.ArgumentsJSON)
			kind := types.ToolKindFromName(call.Name)
			h.sendUpdate(ctx, sid, types.ToolCallStart(
				types.ToolCallID(call.CallID),
				call.Name,
				kind,
				nil,
				rawInput,
			))

			options := []types.PermissionOption{
				{OptionID: "allow", Name: "Allow", Kind: types.PermissionOptionKindAllowOnce},
				{OptionID: "reject", Name: "Reject", Kind: types.PermissionOptionKindRejectOnce},
			}

			permResp, err := h.requestPermission(ctx, sid, call, kind, options)
			if err != nil {
				sess.sendCommand(agent.Command{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: "permission request failed"})
				continue
			}

			if outcomeIsAllowed(permResp.Outcome, options) {
				sess.sendCommand(agent.Command{Type: agent.CmdApproveTool, CallID: call.CallID})
			} else {
				reason := "denied by user"
				if permResp.Outcome.Outcome == "cancelled" {
					reason = "cancelled"
				}
				sess.sendCommand(agent.Command{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: reason})
			}
		}
		return types.PromptResponse{}, false, nil

	case agent.EvtToolStarted:
		h.sendUpdate(ctx, sid, types.ToolCallUpdate(types.ToolCallID(event.CallID), types.ToolCallStatusInProgress, nil, nil))
		return types.PromptResponse{}, false, nil

	case agent.EvtToolCompleted:
		content := []types.ToolCallContent{types.TextToolContent(event.Output)}
		h.sendUpdate(ctx, sid, types.ToolCallUpdate(types.ToolCallID(event.CallID), types.ToolCallStatusCompleted, content, map[string]any{"content": event.Output}))
		return types.PromptResponse{}, false, nil

	case agent.EvtToolFailed:
		content := []types.ToolCallContent{types.TextToolContent(event.Error)}
		h.sendUpdate(ctx, sid, types.ToolCallUpdate(types.ToolCallID(event.CallID), types.ToolCallStatusFailed, content, map[string]any{"error": event.Error}))
		return types.PromptResponse{}, false, nil

	case agent.EvtToolsCompleted:
		return types.PromptResponse{}, false, nil

	case agent.EvtRunCompleted:
		return types.PromptResponse{StopReason: types.StopReasonEndTurn}, true, nil

	case agent.EvtRunCancelled:
		return types.PromptResponse{StopReason: types.StopReasonCancelled}, true, nil

	case agent.EvtRunFailed:
		return types.PromptResponse{}, true, NewRPCError(types.ErrInternalError, event.Error)
	}

	return types.PromptResponse{}, false, nil
}

func (h *Handler) requestPermission(ctx context.Context, sid types.SessionID, call agent.ProposedToolCall, kind types.ToolKind, options []types.PermissionOption) (types.RequestPermissionResponse, error) {
	status := types.ToolCallStatusPending

	var previewData any
	if call.Preview != "" {
		if err := json.Unmarshal([]byte(call.Preview), &previewData); err != nil {
			previewData = nil
		}
	}

	req := types.RequestPermissionRequest{
		SessionID: sid,
		ToolCall: types.ToolCallDetail{
			ToolCallID: types.ToolCallID(call.CallID),
			Title:      &call.Name,
			Kind:       &kind,
			Status:     &status,
			RawInput:   parseRawInput(call.ArgumentsJSON),
			Preview:    previewData,
		},
		Options: options,
	}

	result, err := h.conn.Request(ctx, types.MethodRequestPermission, req)
	if err != nil {
		return types.RequestPermissionResponse{}, fmt.Errorf("protocol: request permission: %w", err)
	}

	var resp types.RequestPermissionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return types.RequestPermissionResponse{}, fmt.Errorf("protocol: unmarshal permission response: %w", err)
	}
	return resp, nil
}

type sessionState struct {
	mu        sync.RWMutex
	agentName string
	sessionID core.SessionID
	cwd       string
	commandCh chan<- agent.Command
	cancelFn  context.CancelFunc
	doneCh    chan struct{}
}

func (s *sessionState) sendCommand(cmd agent.Command) bool {
	s.mu.RLock()
	ch := s.commandCh
	done := s.doneCh
	s.mu.RUnlock()

	if ch == nil {
		return false
	}
	select {
	case ch <- cmd:
		return true
	case <-done:
		return false
	}
}

func outcomeIsAllowed(outcome types.PermissionOutcome, options []types.PermissionOption) bool {
	if outcome.Outcome != "selected" {
		return false
	}
	for _, opt := range options {
		if opt.OptionID == outcome.OptionID {
			return opt.Kind.IsAllow()
		}
	}
	return false
}

// handleCancel stops the active agent run in a session.
//
// ACP: "session/cancel" (notification — no response)
// Params: [types.CancelNotification] — the session ID to cancel.
func (h *Handler) handleCancel(params json.RawMessage) {
	var notif types.CancelNotification
	if err := json.Unmarshal(params, &notif); err != nil {
		return
	}

	val, ok := h.sessions.Load(notif.SessionID)
	if !ok {
		return
	}
	sess := val.(*sessionState)

	sess.sendCommand(agent.Command{Type: agent.CmdCancel})
}

func (h *Handler) sendUpdate(ctx context.Context, sid types.SessionID, update any) {
	_ = h.conn.Notify(ctx, types.MethodSessionUpdate, types.SessionNotification{
		SessionID: sid,
		Update:    update,
	})
}

func extractText(blocks []types.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func parseRawInput(argumentsJSON string) any {
	if argumentsJSON == "" {
		return nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return nil
	}
	return args
}

func parseSkillInvocation(text string) (name string, args string) {
	text = strings.TrimPrefix(text, "/")
	idx := strings.IndexByte(text, ' ')
	if idx < 0 {
		return text, ""
	}
	return text[:idx], strings.TrimSpace(text[idx+1:])
}

func hasACPTools(caps types.ClientCapabilities) bool {
	if caps.Terminal {
		return true
	}
	if caps.Fs != nil && (caps.Fs.ReadTextFile || caps.Fs.WriteTextFile) {
		return true
	}
	return false
}
