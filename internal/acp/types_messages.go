package acp

// ContentBlock represents a typed content element within a prompt or response.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// TextBlock creates a ContentBlock with type "text" and the given text.
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// PromptRequest is a client request to send a user prompt to a session.
type PromptRequest struct {
	SessionID SessionID      `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// StopReason indicates why the agent stopped generating a response.
type StopReason string

const (
	// StopReasonEndTurn indicates the agent completed its response normally.
	StopReasonEndTurn StopReason = "end_turn"
	// StopReasonMaxTokens indicates the agent stopped because the token limit was reached.
	StopReasonMaxTokens StopReason = "max_tokens"
	// StopReasonMaxTurnRequests indicates the agent stopped because the maximum turn requests were reached.
	StopReasonMaxTurnRequests StopReason = "max_turn_requests"
	// StopReasonRefusal indicates the agent refused to respond.
	StopReasonRefusal StopReason = "refusal"
	// StopReasonCancelled indicates the prompt was cancelled by the client.
	StopReasonCancelled StopReason = "cancelled"
)

// PromptResponse is the server's final response after processing a prompt.
type PromptResponse struct {
	StopReason StopReason `json:"stopReason"`
}

// CancelNotification is a client notification to cancel an active prompt.
type CancelNotification struct {
	SessionID SessionID `json:"sessionId"`
}

// SessionNotification is a server notification carrying a session update event.
type SessionNotification struct {
	SessionID SessionID `json:"sessionId"`
	Update    any       `json:"update"`
}

// AgentMessageChunk creates a session update payload for a streamed text chunk from the agent.
func AgentMessageChunk(text string) map[string]any {
	return map[string]any{
		"sessionUpdate": "agent_message_chunk",
		"content":       map[string]any{"type": "text", "text": text},
	}
}

// AgentThoughtChunk creates a session update payload for a streamed reasoning chunk from the agent.
func AgentThoughtChunk(text string) map[string]any {
	return map[string]any{
		"sessionUpdate": "agent_thought_chunk",
		"content":       map[string]any{"type": "text", "text": text},
	}
}
