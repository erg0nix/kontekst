package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/erg0nix/kontekst/internal/agent"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/skills"
)

// Handler is the server-side ACP request handler that manages sessions and routes agent events.
type Handler struct {
	runner   agent.Runner
	registry *agent.Registry
	skills   *skills.Registry
	conn     *Connection
	sessions sync.Map
	caps     ClientCapabilities
}

type sessionState struct {
	mu        sync.Mutex
	agentName string
	sessionID core.SessionID
	cwd       string
	commandCh chan<- agent.Command
	cancelFn  context.CancelFunc
}

func (s *sessionState) sendCommand(cmd agent.Command) bool {
	s.mu.Lock()
	ch := s.commandCh
	s.mu.Unlock()

	if ch == nil {
		return false
	}
	ch <- cmd
	return true
}

// NewHandler creates a Handler with the given agent runner, registry, and skills registry.
func NewHandler(runner agent.Runner, registry *agent.Registry, skillsRegistry *skills.Registry) *Handler {
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
func (h *Handler) Dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case MethodInitialize:
		return h.handleInitialize(ctx, params)
	case MethodAuthenticate:
		return AuthenticateResponse{}, nil
	case MethodSessionNew:
		return h.handleNewSession(ctx, params)
	case MethodSessionLoad:
		return h.handleLoadSession(ctx, params)
	case MethodSessionPrompt:
		return h.handlePrompt(ctx, params)
	case MethodSessionCancel:
		h.handleCancel(params)
		return nil, nil
	case MethodSessionSetMode:
		return SetSessionModeResponse{}, nil
	case MethodSessionSetConfig:
		return SetSessionConfigOptionResponse{ConfigOptions: []SessionConfigOption{}}, nil
	default:
		if strings.HasPrefix(method, "_") {
			return nil, NewRPCError(ErrMethodNotFound, fmt.Sprintf("unknown extension: %s", method))
		}
		return nil, NewRPCError(ErrMethodNotFound, fmt.Sprintf("unknown method: %s", method))
	}
}

func (h *Handler) handleInitialize(_ context.Context, params json.RawMessage) (InitializeResponse, error) {
	var req InitializeRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &req); err != nil {
			return InitializeResponse{}, NewRPCError(ErrInvalidParams, err.Error())
		}
	}

	h.caps = req.ClientCapabilities

	return InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		AgentCapabilities: AgentCapabilities{
			LoadSession: true,
		},
		AgentInfo: &Implementation{
			Name:    "kontekst",
			Title:   "Kontekst",
			Version: "0.1.0",
		},
		AuthMethods: []AuthMethod{},
	}, nil
}

func (h *Handler) handleNewSession(_ context.Context, params json.RawMessage) (NewSessionResponse, error) {
	var req NewSessionRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return NewSessionResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	agentName := agentConfig.DefaultAgentName
	if req.Meta != nil {
		if name, ok := req.Meta["agentName"].(string); ok && name != "" {
			agentName = name
		}
	}

	sid := SessionID(core.NewSessionID())

	h.sessions.Store(sid, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(sid),
		cwd:       req.Cwd,
	})

	if h.skills != nil {
		var commands []Command
		for _, s := range h.skills.ModelInvocableSkills() {
			commands = append(commands, Command{Name: s.Name, Description: s.Description})
		}
		if len(commands) > 0 {
			_ = h.conn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
				SessionID: sid,
				Update:    AvailableCommandsUpdate(commands),
			})
		}
	}

	return NewSessionResponse{SessionID: sid}, nil
}

func (h *Handler) handleLoadSession(_ context.Context, params json.RawMessage) (LoadSessionResponse, error) {
	var req LoadSessionRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return LoadSessionResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	agentName := agentConfig.DefaultAgentName

	h.sessions.Store(req.SessionID, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(req.SessionID),
		cwd:       req.Cwd,
	})

	return LoadSessionResponse{SessionID: req.SessionID}, nil
}

func (h *Handler) handlePrompt(ctx context.Context, params json.RawMessage) (PromptResponse, error) {
	var req PromptRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return PromptResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	val, ok := h.sessions.Load(req.SessionID)
	if !ok {
		return PromptResponse{}, NewRPCError(ErrNotFound, "session not found")
	}
	sess := val.(*sessionState)

	promptText := extractText(req.Prompt)

	var skill *skills.Skill
	var skillContent string
	if strings.HasPrefix(promptText, "/") {
		skillName, args := parseSkillInvocation(promptText)
		if h.skills != nil {
			if loadedSkill, found := h.skills.Get(skillName); found {
				rendered, err := loadedSkill.Render(args)
				if err != nil {
					return PromptResponse{}, NewRPCError(ErrInvalidParams, fmt.Sprintf("failed to render skill: %v", err))
				}
				skill = loadedSkill
				skillContent = rendered
				promptText = args
			}
		}
	}

	agentCfg, err := h.registry.Load(sess.agentName)
	if err != nil {
		return PromptResponse{}, NewRPCError(ErrInvalidParams, err.Error())
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
		runCfg.Tools = NewACPToolExecutor(h.conn, req.SessionID, h.caps)
	}

	commandCh, eventCh, err := h.runner.StartRun(runCfg)
	if err != nil {
		cancelFn()
		return PromptResponse{}, NewRPCError(ErrInternalError, err.Error())
	}

	sess.mu.Lock()
	sess.cancelFn = cancelFn
	sess.commandCh = commandCh
	sess.mu.Unlock()

	return h.forwardEvents(runCtx, req.SessionID, sess, eventCh)
}

func (h *Handler) forwardEvents(ctx context.Context, sid SessionID, sess *sessionState, eventCh <-chan agent.Event) (PromptResponse, error) {
	defer func() {
		sess.mu.Lock()
		if sess.cancelFn != nil {
			sess.cancelFn()
		}
		sess.commandCh = nil
		sess.cancelFn = nil
		sess.mu.Unlock()
	}()

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return PromptResponse{StopReason: StopReasonEndTurn}, nil
			}

			resp, done, err := h.processEvent(ctx, sid, sess, event)
			if err != nil {
				return PromptResponse{}, err
			}
			if done {
				return resp, nil
			}

		case <-h.conn.Done():
			sess.sendCommand(agent.Command{Type: agent.CmdCancel})
			return PromptResponse{StopReason: StopReasonCancelled}, nil

		case <-ctx.Done():
			sess.sendCommand(agent.Command{Type: agent.CmdCancel})
			return PromptResponse{StopReason: StopReasonCancelled}, nil
		}
	}
}

func (h *Handler) processEvent(ctx context.Context, sid SessionID, sess *sessionState, event agent.Event) (PromptResponse, bool, error) {
	switch event.Type {
	case agent.EvtRunStarted:
		return PromptResponse{}, false, nil

	case agent.EvtTokenDelta:
		h.sendUpdate(sid, AgentMessageChunk(event.Token))
		return PromptResponse{}, false, nil

	case agent.EvtReasoningDelta:
		h.sendUpdate(sid, AgentThoughtChunk(event.Reasoning))
		return PromptResponse{}, false, nil

	case agent.EvtTurnCompleted:
		if event.Response.Reasoning != "" {
			h.sendUpdate(sid, AgentThoughtChunk(event.Response.Reasoning))
		}
		if event.Response.Content != "" {
			h.sendUpdate(sid, AgentMessageChunk(event.Response.Content))
		}
		if event.Snapshot != nil {
			_ = h.conn.Notify(ctx, MethodKontekstContext, event.Snapshot)
		}
		return PromptResponse{}, false, nil

	case agent.EvtToolsProposed:
		for _, call := range event.Calls {
			rawInput := parseRawInput(call.ArgumentsJSON)
			kind := ToolKindFromName(call.Name)
			h.sendUpdate(sid, ToolCallStart(
				ToolCallID(call.CallID),
				call.Name,
				kind,
				nil,
				rawInput,
			))

			options := []PermissionOption{
				{OptionID: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
				{OptionID: "reject", Name: "Reject", Kind: PermissionOptionKindRejectOnce},
			}

			permResp, err := h.requestPermission(ctx, sid, call, kind, options)
			if err != nil {
				sess.sendCommand(agent.Command{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: "permission request failed"})
				continue
			}

			if isAllowOutcome(permResp.Outcome, options) {
				sess.sendCommand(agent.Command{Type: agent.CmdApproveTool, CallID: call.CallID})
			} else {
				reason := "denied by user"
				if permResp.Outcome.Outcome == "cancelled" {
					reason = "cancelled"
				}
				sess.sendCommand(agent.Command{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: reason})
			}
		}
		return PromptResponse{}, false, nil

	case agent.EvtToolStarted:
		h.sendUpdate(sid, ToolCallUpdate(ToolCallID(event.CallID), ToolCallStatusInProgress, nil, nil))
		return PromptResponse{}, false, nil

	case agent.EvtToolCompleted:
		content := []ToolCallContent{TextToolContent(event.Output)}
		h.sendUpdate(sid, ToolCallUpdate(ToolCallID(event.CallID), ToolCallStatusCompleted, content, map[string]any{"content": event.Output}))
		return PromptResponse{}, false, nil

	case agent.EvtToolFailed:
		content := []ToolCallContent{TextToolContent(event.Error)}
		h.sendUpdate(sid, ToolCallUpdate(ToolCallID(event.CallID), ToolCallStatusFailed, content, map[string]any{"error": event.Error}))
		return PromptResponse{}, false, nil

	case agent.EvtToolsCompleted:
		return PromptResponse{}, false, nil

	case agent.EvtRunCompleted:
		return PromptResponse{StopReason: StopReasonEndTurn}, true, nil

	case agent.EvtRunCancelled:
		return PromptResponse{StopReason: StopReasonCancelled}, true, nil

	case agent.EvtRunFailed:
		return PromptResponse{}, true, NewRPCError(ErrInternalError, event.Error)
	}

	return PromptResponse{}, false, nil
}

func (h *Handler) requestPermission(ctx context.Context, sid SessionID, call agent.ProposedToolCall, kind ToolKind, options []PermissionOption) (RequestPermissionResponse, error) {
	status := ToolCallStatusPending

	var previewData any
	if call.Preview != "" {
		if err := json.Unmarshal([]byte(call.Preview), &previewData); err != nil {
			previewData = nil
		}
	}

	req := RequestPermissionRequest{
		SessionID: sid,
		ToolCall: ToolCallDetail{
			ToolCallID: ToolCallID(call.CallID),
			Title:      &call.Name,
			Kind:       &kind,
			Status:     &status,
			RawInput:   parseRawInput(call.ArgumentsJSON),
			Preview:    previewData,
		},
		Options: options,
	}

	result, err := h.conn.Request(ctx, MethodRequestPermission, req)
	if err != nil {
		return RequestPermissionResponse{}, fmt.Errorf("acp: request permission: %w", err)
	}

	var resp RequestPermissionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return RequestPermissionResponse{}, fmt.Errorf("acp: unmarshal permission response: %w", err)
	}
	return resp, nil
}

func isAllowOutcome(outcome PermissionOutcome, options []PermissionOption) bool {
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

func (h *Handler) handleCancel(params json.RawMessage) {
	var notif CancelNotification
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

func (h *Handler) sendUpdate(sid SessionID, update any) {
	_ = h.conn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
		SessionID: sid,
		Update:    update,
	})
}

func extractText(blocks []ContentBlock) string {
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

func hasACPTools(caps ClientCapabilities) bool {
	if caps.Terminal {
		return true
	}
	if caps.Fs != nil && (caps.Fs.ReadTextFile || caps.Fs.WriteTextFile) {
		return true
	}
	return false
}
