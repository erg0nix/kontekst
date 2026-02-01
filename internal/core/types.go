package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
	AgentName  string      `json:"agent_name,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ToolResult struct {
	CallID  string `json:"call_id"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error,omitempty"`
}

type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ChatResponse struct {
	Content   string
	Reasoning string
	ToolCalls []ToolCall
}

type SamplingConfig struct {
	Temperature   *float64
	TopP          *float64
	TopK          *int
	RepeatPenalty *float64
	MaxTokens     *int
}

type SkillMetadata struct {
	Name string
	Path string
}

type RunID string

type SessionID string

func NewRunID() RunID {
	return RunID(newID("run"))
}

func NewSessionID() SessionID {
	return SessionID(newID("sess"))
}

func newID(prefix string) string {
	buffer := make([]byte, 6)
	_, _ = rand.Read(buffer)
	seed := hex.EncodeToString(buffer)
	timestamp := time.Now().UTC().Format("20060102T150405.000000000")

	return prefix + "_" + timestamp + "_" + seed
}
