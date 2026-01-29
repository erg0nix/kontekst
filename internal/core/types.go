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
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolResult *ToolResult
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type ToolResult struct {
	CallID  string
	Name    string
	Output  string
	IsError bool
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
