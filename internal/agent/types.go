package agent

import "github.com/erg0nix/kontekst/internal/core"

type CommandType string

type EventType string

const (
	CmdApproveTool    CommandType = "approve_tool"
	CmdDenyTool       CommandType = "deny_tool"
	CmdCancel         CommandType = "cancel"
	EvtRunStarted     EventType   = "run_started"
	EvtTokenDelta     EventType   = "token_delta"
	EvtReasoningDelta EventType   = "reasoning_delta"
	EvtTurnCompleted  EventType   = "turn_completed"
	EvtToolsProposed  EventType   = "tools_proposed"
	EvtToolStarted    EventType   = "tool_execution_started"
	EvtToolCompleted  EventType   = "tool_execution_completed"
	EvtToolFailed     EventType   = "tool_execution_failed"
	EvtToolsCompleted EventType   = "tools_completed"
	EvtRunCompleted   EventType   = "run_completed"
	EvtRunCancelled   EventType   = "run_cancelled"
	EvtRunFailed      EventType   = "run_failed"
)

type Command struct {
	Type   CommandType
	CallID string
	Reason string
}

type Event struct {
	Type      EventType
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
