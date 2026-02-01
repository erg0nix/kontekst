package agent

import "github.com/erg0nix/kontekst/internal/core"

type AgentCommandType string

type AgentEventType string

const (
	CmdApproveTool        AgentCommandType = "approve_tool"
	CmdDenyTool           AgentCommandType = "deny_tool"
	CmdApproveAll         AgentCommandType = "approve_all"
	CmdDenyAll            AgentCommandType = "deny_all"
	CmdCancel             AgentCommandType = "cancel"
	EvtRunStarted         AgentEventType   = "run_started"
	EvtTokenDelta         AgentEventType   = "token_delta"
	EvtReasoningDelta     AgentEventType   = "reasoning_delta"
	EvtToolBatch          AgentEventType   = "tool_batch_proposed"
	EvtToolStarted        AgentEventType   = "tool_execution_started"
	EvtToolCompleted      AgentEventType   = "tool_execution_completed"
	EvtToolFailed         AgentEventType   = "tool_execution_failed"
	EvtToolBatchCompleted AgentEventType   = "tool_batch_completed"
	EvtRunCompleted       AgentEventType   = "run_completed"
	EvtRunCancelled       AgentEventType   = "run_cancelled"
	EvtRunFailed          AgentEventType   = "run_failed"
)

type AgentCommand struct {
	Type    AgentCommandType
	CallID  string
	BatchID string
	Reason  string
}

type AgentEvent struct {
	Type      AgentEventType
	RunID     core.RunID
	SessionID core.SessionID
	AgentName string
	Token     string
	Reasoning string
	BatchID   string
	Calls     []ProposedToolCall
	CallID    string
	Output    string
	Response  core.ChatResponse
	Error     string
}

type ProposedToolCall struct {
	CallID        string
	Name          string
	ArgumentsJSON string
	Preview       string
}
