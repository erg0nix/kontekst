package acp

import (
	"encoding/json"
	"testing"
)

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return string(data)
}

func mustUnmarshalMap(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return m
}

func TestNewSessionRequest_McpServersEmptyArray(t *testing.T) {
	req := NewSessionRequest{
		Cwd:        "/home/user/project",
		McpServers: []McpServer{},
	}

	data := mustMarshal(t, req)
	m := mustUnmarshalMap(t, data)

	servers, ok := m["mcpServers"].([]any)
	if !ok {
		t.Fatal("mcpServers not an array")
	}
	if len(servers) != 0 {
		t.Errorf("mcpServers length = %d, want 0", len(servers))
	}

	if m["cwd"] != "/home/user/project" {
		t.Errorf("cwd = %v, want /home/user/project", m["cwd"])
	}
}

func TestNewSessionRequest_WithMeta(t *testing.T) {
	req := NewSessionRequest{
		Cwd:        "/tmp",
		McpServers: []McpServer{},
		Meta:       map[string]any{"agentName": "myagent"},
	}

	data := mustMarshal(t, req)
	m := mustUnmarshalMap(t, data)

	meta, ok := m["_meta"].(map[string]any)
	if !ok {
		t.Fatal("_meta not a map")
	}
	if meta["agentName"] != "myagent" {
		t.Errorf("_meta.agentName = %v, want myagent", meta["agentName"])
	}
}

func TestAgentMessageChunk(t *testing.T) {
	update := AgentMessageChunk("hello world")
	notif := SessionNotification{
		SessionID: "sess_1",
		Update:    update,
	}

	data := mustMarshal(t, notif)
	m := mustUnmarshalMap(t, data)

	if m["sessionId"] != "sess_1" {
		t.Errorf("sessionId = %v, want sess_1", m["sessionId"])
	}

	u, ok := m["update"].(map[string]any)
	if !ok {
		t.Fatal("update not a map")
	}
	if u["sessionUpdate"] != "agent_message_chunk" {
		t.Errorf("sessionUpdate = %v, want agent_message_chunk", u["sessionUpdate"])
	}

	content, ok := u["content"].(map[string]any)
	if !ok {
		t.Fatal("content not a map")
	}
	if content["type"] != "text" {
		t.Errorf("content.type = %v, want text", content["type"])
	}
	if content["text"] != "hello world" {
		t.Errorf("content.text = %v, want hello world", content["text"])
	}
}

func TestToolCallStart(t *testing.T) {
	update := ToolCallStart(
		"call_1",
		"Reading main.go",
		ToolKindRead,
		[]ToolCallLocation{{Path: "/project/main.go"}},
		map[string]any{"path": "/project/main.go"},
	)

	notif := SessionNotification{SessionID: "sess_1", Update: update}
	data := mustMarshal(t, notif)
	m := mustUnmarshalMap(t, data)

	u := m["update"].(map[string]any)
	if u["sessionUpdate"] != "tool_call" {
		t.Errorf("sessionUpdate = %v, want tool_call", u["sessionUpdate"])
	}
	if u["toolCallId"] != "call_1" {
		t.Errorf("toolCallId = %v, want call_1", u["toolCallId"])
	}
	if u["kind"] != "read" {
		t.Errorf("kind = %v, want read", u["kind"])
	}
	if u["status"] != "pending" {
		t.Errorf("status = %v, want pending", u["status"])
	}

	content, ok := u["content"].([]any)
	if !ok {
		t.Fatal("content not an array")
	}
	if len(content) != 0 {
		t.Errorf("content length = %d, want 0", len(content))
	}

	locs := u["locations"].([]any)
	if len(locs) != 1 {
		t.Fatalf("locations length = %d, want 1", len(locs))
	}
	loc := locs[0].(map[string]any)
	if loc["path"] != "/project/main.go" {
		t.Errorf("location.path = %v, want /project/main.go", loc["path"])
	}
}

func TestToolKindFromName(t *testing.T) {
	tests := []struct {
		name string
		want ToolKind
	}{
		{"read_file", ToolKindRead},
		{"write_file", ToolKindEdit},
		{"edit_file", ToolKindEdit},
		{"list_files", ToolKindSearch},
		{"run_command", ToolKindExecute},
		{"web_fetch", ToolKindFetch},
		{"skill", ToolKindOther},
		{"unknown_tool", ToolKindOther},
	}

	for _, tt := range tests {
		got := ToolKindFromName(tt.name)
		if got != tt.want {
			t.Errorf("ToolKindFromName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPermissionResponseSelected(t *testing.T) {
	resp := RequestPermissionResponse{
		Outcome: PermissionSelected("allow"),
	}

	data := mustMarshal(t, resp)
	m := mustUnmarshalMap(t, data)

	outcome := m["outcome"].(map[string]any)
	if outcome["outcome"] != "selected" {
		t.Errorf("outcome = %v, want selected", outcome["outcome"])
	}
	if outcome["optionId"] != "allow" {
		t.Errorf("optionId = %v, want allow", outcome["optionId"])
	}
}

func TestPermissionResponseCancelled(t *testing.T) {
	resp := RequestPermissionResponse{
		Outcome: PermissionCancelled(),
	}

	data := mustMarshal(t, resp)
	m := mustUnmarshalMap(t, data)

	outcome := m["outcome"].(map[string]any)
	if outcome["outcome"] != "cancelled" {
		t.Errorf("outcome = %v, want cancelled", outcome["outcome"])
	}
	if _, ok := outcome["optionId"]; ok {
		t.Error("optionId should not be present for cancelled")
	}
}

func TestPermissionOptionKindIsAllow(t *testing.T) {
	tests := []struct {
		kind PermissionOptionKind
		want bool
	}{
		{PermissionOptionKindAllowOnce, true},
		{PermissionOptionKindAllowAlways, true},
		{PermissionOptionKindRejectOnce, false},
		{PermissionOptionKindRejectAlways, false},
	}

	for _, tt := range tests {
		if got := tt.kind.IsAllow(); got != tt.want {
			t.Errorf("%q.IsAllow() = %v, want %v", tt.kind, got, tt.want)
		}
	}
}
