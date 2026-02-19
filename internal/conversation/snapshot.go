package conversation

import "github.com/erg0nix/kontekst/internal/core"

// Snapshot captures the token and message budget state of a conversation context at a point in time.
type Snapshot struct {
	ContextSize     int            `json:"context_size"`
	SystemTokens    int            `json:"system_tokens"`
	ToolTokens      int            `json:"tool_tokens"`
	HistoryTokens   int            `json:"history_tokens"`
	MemoryTokens    int            `json:"memory_tokens"`
	TotalTokens     int            `json:"total_tokens"`
	RemainingTokens int            `json:"remaining_tokens"`
	HistoryMessages int            `json:"history_messages"`
	MemoryMessages  int            `json:"memory_messages"`
	TotalMessages   int            `json:"total_messages"`
	HistoryBudget   int            `json:"history_budget"`
	Messages        []MessageStats `json:"messages,omitempty"`
}

// MessageStats holds token count and source metadata for a single message in a context snapshot.
type MessageStats struct {
	Role   core.Role `json:"role"`
	Tokens int       `json:"tokens"`
	Source string    `json:"source"`
}
