package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type RunID string

type SessionID string

type ToolCallID string

type RequestID string

func NewRunID() RunID {
	return RunID("run_" + timestamp() + "_" + randomSeed())
}

func NewSessionID() SessionID {
	return SessionID("sess_" + timestamp() + "_" + randomSeed())
}

func NewToolCallID() ToolCallID {
	return ToolCallID("call_" + randomSeed())
}

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
