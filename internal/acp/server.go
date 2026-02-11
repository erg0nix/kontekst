package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/erg0nix/kontekst/internal/agent"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/skills"
)

type Handler struct {
	Runner   agent.Runner
	Registry *agent.Registry
	Skills   *skills.Registry
	conn     *Connection
	sessions sync.Map
}

type sessionState struct {
	agentName string
	sessionID core.SessionID
	cwd       string
	commandCh chan<- agent.AgentCommand
	cancelFn  context.CancelFunc
}

func NewHandler(runner agent.Runner, registry *agent.Registry, skillsRegistry *skills.Registry) *Handler {
	return &Handler{
		Runner:   runner,
		Registry: registry,
		Skills:   skillsRegistry,
	}
}

func (h *Handler) SetConnection(conn *Connection) {
	h.conn = conn
}

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
	case MethodSessionSetCfg:
		return SetSessionConfigOptionResponse{}, nil
	default:
		if strings.HasPrefix(method, "_") {
			return nil, NewRPCError(ErrMethodNotFound, fmt.Sprintf("unknown extension: %s", method))
		}
		return nil, NewRPCError(ErrMethodNotFound, fmt.Sprintf("unknown method: %s", method))
	}
}

func (h *Handler) handleInitialize(_ context.Context, _ json.RawMessage) (InitializeResponse, error) {
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

	sid := SessionId(core.NewSessionID())

	h.sessions.Store(sid, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(sid),
		cwd:       req.Cwd,
	})

	if h.Skills != nil {
		var commands []Command
		for _, s := range h.Skills.ModelInvocableSkills() {
			commands = append(commands, Command{Name: s.Name, Description: s.Description})
		}
		if len(commands) > 0 {
			_ = h.conn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
				SessionId: sid,
				Update:    AvailableCommandsUpdate(commands),
			})
		}
	}

	return NewSessionResponse{SessionId: sid}, nil
}

func (h *Handler) handleLoadSession(_ context.Context, params json.RawMessage) (LoadSessionResponse, error) {
	var req LoadSessionRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return LoadSessionResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	agentName := agentConfig.DefaultAgentName

	h.sessions.Store(req.SessionId, &sessionState{
		agentName: agentName,
		sessionID: core.SessionID(req.SessionId),
	})

	return LoadSessionResponse{SessionId: req.SessionId}, nil
}

func (h *Handler) handlePrompt(ctx context.Context, params json.RawMessage) (PromptResponse, error) {
	var req PromptRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return PromptResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	val, ok := h.sessions.Load(req.SessionId)
	if !ok {
		return PromptResponse{}, NewRPCError(ErrNotFound, "session not found")
	}
	sess := val.(*sessionState)

	promptText := extractText(req.Prompt)

	var skill *skills.Skill
	var skillContent string
	if strings.HasPrefix(promptText, "/") {
		skillName, args := parseSkillInvocation(promptText)
		if h.Skills != nil {
			if loadedSkill, found := h.Skills.Get(skillName); found {
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

	agentCfg, err := h.Registry.Load(sess.agentName)
	if err != nil {
		return PromptResponse{}, NewRPCError(ErrInvalidParams, err.Error())
	}

	runCtx, cancelFn := context.WithCancel(ctx)
	sess.cancelFn = cancelFn

	commandCh, eventCh, err := h.Runner.StartRun(agent.RunConfig{
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
	})
	if err != nil {
		cancelFn()
		return PromptResponse{}, NewRPCError(ErrInternalError, err.Error())
	}

	sess.commandCh = commandCh
	return h.forwardEvents(runCtx, req.SessionId, sess, eventCh)
}

func (h *Handler) forwardEvents(ctx context.Context, sid SessionId, sess *sessionState, eventCh <-chan agent.AgentEvent) (PromptResponse, error) {
	defer func() {
		if sess.cancelFn != nil {
			sess.cancelFn()
		}
		sess.commandCh = nil
		sess.cancelFn = nil
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
			if sess.commandCh != nil {
				sess.commandCh <- agent.AgentCommand{Type: agent.CmdCancel}
			}
			return PromptResponse{StopReason: StopReasonCancelled}, nil

		case <-ctx.Done():
			return PromptResponse{StopReason: StopReasonCancelled}, nil
		}
	}
}

func (h *Handler) processEvent(ctx context.Context, sid SessionId, sess *sessionState, event agent.AgentEvent) (PromptResponse, bool, error) {
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
			var rawInput any
			if call.ArgumentsJSON != "" {
				var args map[string]any
				if err := json.Unmarshal([]byte(call.ArgumentsJSON), &args); err == nil {
					rawInput = args
				}
			}

			kind := ToolKindFromName(call.Name)
			h.sendUpdate(sid, ToolCallStart(
				ToolCallId(call.CallID),
				call.Name,
				kind,
				nil,
				rawInput,
			))

			permResp, err := h.requestPermission(ctx, sid, call, kind)
			if err != nil {
				if sess.commandCh != nil {
					sess.commandCh <- agent.AgentCommand{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: "permission request failed"}
				}
				continue
			}

			if permResp.Outcome.Selected != nil && permResp.Outcome.Selected.OptionId == "allow" {
				sess.commandCh <- agent.AgentCommand{Type: agent.CmdApproveTool, CallID: call.CallID}
			} else {
				reason := "denied by user"
				if permResp.Outcome.Cancelled != nil {
					reason = "cancelled"
				}
				sess.commandCh <- agent.AgentCommand{Type: agent.CmdDenyTool, CallID: call.CallID, Reason: reason}
			}
		}
		return PromptResponse{}, false, nil

	case agent.EvtToolStarted:
		h.sendUpdate(sid, ToolCallUpdate(ToolCallId(event.CallID), ToolCallStatusInProgress, nil, nil))
		return PromptResponse{}, false, nil

	case agent.EvtToolCompleted:
		content := []ToolCallContent{TextToolContent(event.Output)}
		h.sendUpdate(sid, ToolCallUpdate(ToolCallId(event.CallID), ToolCallStatusCompleted, content, map[string]any{"content": event.Output}))
		return PromptResponse{}, false, nil

	case agent.EvtToolFailed:
		content := []ToolCallContent{TextToolContent(event.Error)}
		h.sendUpdate(sid, ToolCallUpdate(ToolCallId(event.CallID), ToolCallStatusFailed, content, map[string]any{"error": event.Error}))
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

func (h *Handler) requestPermission(ctx context.Context, sid SessionId, call agent.ProposedToolCall, kind ToolKind) (RequestPermissionResponse, error) {
	status := ToolCallStatusPending
	var rawInput any
	if call.ArgumentsJSON != "" {
		var args map[string]any
		if err := json.Unmarshal([]byte(call.ArgumentsJSON), &args); err == nil {
			rawInput = args
		}
	}

	req := RequestPermissionRequest{
		SessionId: sid,
		ToolCall: ToolCallDetail{
			ToolCallId: ToolCallId(call.CallID),
			Title:      &call.Name,
			Kind:       &kind,
			Status:     &status,
			RawInput:   rawInput,
		},
		Options: []PermissionOption{
			{OptionId: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
			{OptionId: "reject", Name: "Reject", Kind: PermissionOptionKindRejectOnce},
		},
	}

	result, err := h.conn.Request(ctx, MethodRequestPerm, req)
	if err != nil {
		return RequestPermissionResponse{}, err
	}

	var resp RequestPermissionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return RequestPermissionResponse{}, fmt.Errorf("acp: unmarshal permission response: %w", err)
	}
	return resp, nil
}

func (h *Handler) handleCancel(params json.RawMessage) {
	var notif CancelNotification
	if err := json.Unmarshal(params, &notif); err != nil {
		return
	}

	val, ok := h.sessions.Load(notif.SessionId)
	if !ok {
		return
	}
	sess := val.(*sessionState)

	if sess.commandCh != nil {
		sess.commandCh <- agent.AgentCommand{Type: agent.CmdCancel}
	}
}

func (h *Handler) sendUpdate(sid SessionId, update any) {
	_ = h.conn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
		SessionId: sid,
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

func parseSkillInvocation(text string) (name string, args string) {
	text = strings.TrimPrefix(text, "/")
	idx := strings.IndexByte(text, ' ')
	if idx < 0 {
		return text, ""
	}
	return text[:idx], strings.TrimSpace(text[idx+1:])
}
