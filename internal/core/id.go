package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// RunID uniquely identifies an agent run within a session.
type RunID string

// SessionID uniquely identifies a conversation session.
type SessionID string

// ToolCallID uniquely identifies a single tool invocation.
type ToolCallID string

// RequestID uniquely identifies a JSON-RPC request.
type RequestID string

// NewRunID generates a new RunID with a timestamp and random suffix.
func NewRunID() RunID {
	return RunID("run_" + timestamp() + "_" + randomSeed())
}

// NewSessionID generates a new SessionID with a timestamp and random suffix.
func NewSessionID() SessionID {
	return SessionID("sess_" + timestamp() + "_" + randomSeed())
}

// NewToolCallID generates a new ToolCallID with a random suffix.
func NewToolCallID() ToolCallID {
	return ToolCallID("call_" + randomSeed())
}

// NewRequestID generates a new RequestID with a timestamp and random suffix.
func NewRequestID() RequestID {
	return RequestID("req_" + timestamp() + "_" + randomSeed())
}

func timestamp() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}

func randomSeed() string {
	buffer := make([]byte, 6)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}
