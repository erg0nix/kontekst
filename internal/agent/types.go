package agent

import "github.com/erg0nix/kontekst/internal/core"

// CommandType identifies the kind of command sent from a client to the agent.
type CommandType string

// EventType identifies the kind of event emitted by the agent during a run.
type EventType string

const (
	// CmdApproveTool is a command that approves a proposed tool call for execution.
	CmdApproveTool CommandType = "approve_tool"
	// CmdDenyTool is a command that denies a proposed tool call.
	CmdDenyTool CommandType = "deny_tool"
	// CmdCancel is a command that cancels the current agent run.
	CmdCancel CommandType = "cancel"
	// EvtRunStarted is emitted when a new agent run begins.
	EvtRunStarted EventType = "run_started"
	// EvtTokenDelta is emitted for each streamed token from the LLM.
	EvtTokenDelta EventType = "token_delta"
	// EvtReasoningDelta is emitted for each streamed reasoning token from the LLM.
	EvtReasoningDelta EventType = "reasoning_delta"
	// EvtTurnCompleted is emitted after the LLM finishes generating a response.
	EvtTurnCompleted EventType = "turn_completed"
	// EvtToolsProposed is emitted when the LLM proposes one or more tool calls for approval.
	EvtToolsProposed EventType = "tools_proposed"
	// EvtToolStarted is emitted when a tool begins executing.
	EvtToolStarted EventType = "tool_execution_started"
	// EvtToolCompleted is emitted when a tool finishes executing successfully.
	EvtToolCompleted EventType = "tool_execution_completed"
	// EvtToolFailed is emitted when a tool execution fails with an error.
	EvtToolFailed EventType = "tool_execution_failed"
	// EvtToolsCompleted is emitted after all tool calls in a batch have finished.
	EvtToolsCompleted EventType = "tools_completed"
	// EvtRunCompleted is emitted when the agent run finishes successfully.
	EvtRunCompleted EventType = "run_completed"
	// EvtRunCancelled is emitted when the agent run is cancelled by the client.
	EvtRunCancelled EventType = "run_cancelled"
	// EvtRunFailed is emitted when the agent run terminates due to an error.
	EvtRunFailed EventType = "run_failed"
)

// Command represents an instruction sent from a client to control the agent run.
type Command struct {
	Type   CommandType
	CallID string
	Reason string
}

// Event represents a notification emitted by the agent during a run.
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

// ProposedToolCall represents a tool call proposed by the LLM that awaits client approval.
type ProposedToolCall struct {
	CallID        string
	Name          string
	ArgumentsJSON string
	Preview       string
}
