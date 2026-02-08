package agent

import "github.com/erg0nix/kontekst/internal/core"

type AgentCommandType string

type AgentEventType string

const (
	CmdApproveTool     AgentCommandType = "approve_tool"
	CmdDenyTool        AgentCommandType = "deny_tool"
	CmdCancel          AgentCommandType = "cancel"
	EvtRunStarted      AgentEventType   = "run_started"
	EvtTokenDelta      AgentEventType   = "token_delta"
	EvtReasoningDelta  AgentEventType   = "reasoning_delta"
	EvtTurnCompleted   AgentEventType   = "turn_completed"
	EvtToolsProposed   AgentEventType   = "tools_proposed"
	EvtToolStarted     AgentEventType   = "tool_execution_started"
	EvtToolCompleted   AgentEventType   = "tool_execution_completed"
	EvtToolFailed      AgentEventType   = "tool_execution_failed"
	EvtToolsCompleted  AgentEventType   = "tools_completed"
	EvtRunCompleted    AgentEventType   = "run_completed"
	EvtRunCancelled    AgentEventType   = "run_cancelled"
	EvtRunFailed       AgentEventType   = "run_failed"
	EvtContextSnapshot AgentEventType   = "context_snapshot"
)

type AgentCommand struct {
	Type   AgentCommandType
	CallID string
	Reason string
}

type AgentEvent struct {
	Type      AgentEventType
	RunID     core.RunID
	SessionID core.SessionID
	AgentName string
	Token     string
	Reasoning string
	Calls     []ProposedToolCall
	CallID    string
	Output    string
	Response  core.ChatResponse
	Snapshot  *core.ContextSnapshot
	Error     string
}

type ProposedToolCall struct {
	CallID        string
	Name          string
	ArgumentsJSON string
	Preview       string
}
