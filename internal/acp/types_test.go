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

func TestInitializeResponse(t *testing.T) {
	resp := InitializeResponse{
		ProtocolVersion: 1,
		AgentCapabilities: AgentCapabilities{
			LoadSession: true,
		},
		AgentInfo: &Implementation{
			Name:    "kontekst",
			Title:   "Kontekst",
			Version: "0.1.0",
		},
		AuthMethods: []AuthMethod{},
	}

	data := mustMarshal(t, resp)
	m := mustUnmarshalMap(t, data)

	if m["protocolVersion"] != float64(1) {
		t.Errorf("protocolVersion = %v, want 1", m["protocolVersion"])
	}

	caps, ok := m["agentCapabilities"].(map[string]any)
	if !ok {
		t.Fatal("agentCapabilities not a map")
	}
	if caps["loadSession"] != true {
		t.Errorf("loadSession = %v, want true", caps["loadSession"])
	}

	info, ok := m["agentInfo"].(map[string]any)
	if !ok {
		t.Fatal("agentInfo not a map")
	}
	if info["name"] != "kontekst" {
		t.Errorf("agentInfo.name = %v, want kontekst", info["name"])
	}

	methods, ok := m["authMethods"].([]any)
	if !ok {
		t.Fatal("authMethods not an array")
	}
	if len(methods) != 0 {
		t.Errorf("authMethods length = %d, want 0", len(methods))
	}
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
		SessionId: "sess_1",
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

func TestAgentThoughtChunk(t *testing.T) {
	update := AgentThoughtChunk("thinking...")

	data := mustMarshal(t, update)
	m := mustUnmarshalMap(t, data)

	if m["sessionUpdate"] != "agent_thought_chunk" {
		t.Errorf("sessionUpdate = %v, want agent_thought_chunk", m["sessionUpdate"])
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

	notif := SessionNotification{SessionId: "sess_1", Update: update}
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

func TestToolCallUpdate(t *testing.T) {
	update := ToolCallUpdate(
		"call_1",
		ToolCallStatusCompleted,
		[]ToolCallContent{TextToolContent("file contents...")},
		map[string]any{"content": "file contents..."},
	)

	data := mustMarshal(t, update)
	m := mustUnmarshalMap(t, data)

	if m["sessionUpdate"] != "tool_call_update" {
		t.Errorf("sessionUpdate = %v, want tool_call_update", m["sessionUpdate"])
	}
	if m["toolCallId"] != "call_1" {
		t.Errorf("toolCallId = %v, want call_1", m["toolCallId"])
	}
	if m["status"] != "completed" {
		t.Errorf("status = %v, want completed", m["status"])
	}

	content := m["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("content length = %d, want 1", len(content))
	}

	c := content[0].(map[string]any)
	if c["type"] != "content" {
		t.Errorf("content[0].type = %v, want content", c["type"])
	}
}

func TestRequestPermissionRoundTrip(t *testing.T) {
	kind := ToolKindEdit
	status := ToolCallStatusPending
	req := RequestPermissionRequest{
		SessionId: "sess_1",
		ToolCall: ToolCallDetail{
			ToolCallId: "call_1",
			Title:      strPtr("Write config.json"),
			Kind:       &kind,
			Status:     &status,
			RawInput:   map[string]any{"path": "/project/config.json"},
		},
		Options: []PermissionOption{
			{OptionId: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
			{OptionId: "reject", Name: "Reject", Kind: PermissionOptionKindRejectOnce},
		},
	}

	data := mustMarshal(t, req)

	var decoded RequestPermissionRequest
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.SessionId != "sess_1" {
		t.Errorf("sessionId = %v, want sess_1", decoded.SessionId)
	}
	if decoded.ToolCall.ToolCallId != "call_1" {
		t.Errorf("toolCallId = %v, want call_1", decoded.ToolCall.ToolCallId)
	}
	if *decoded.ToolCall.Title != "Write config.json" {
		t.Errorf("title = %v, want Write config.json", *decoded.ToolCall.Title)
	}
	if len(decoded.Options) != 2 {
		t.Fatalf("options length = %d, want 2", len(decoded.Options))
	}
	if decoded.Options[0].Kind != PermissionOptionKindAllowOnce {
		t.Errorf("options[0].kind = %v, want allow_once", decoded.Options[0].Kind)
	}
}

func TestPromptRequest(t *testing.T) {
	req := PromptRequest{
		SessionId: "sess_1",
		Prompt:    []ContentBlock{TextBlock("Fix the bug in auth.go")},
	}

	data := mustMarshal(t, req)
	m := mustUnmarshalMap(t, data)

	if m["sessionId"] != "sess_1" {
		t.Errorf("sessionId = %v, want sess_1", m["sessionId"])
	}

	prompt := m["prompt"].([]any)
	if len(prompt) != 1 {
		t.Fatalf("prompt length = %d, want 1", len(prompt))
	}

	block := prompt[0].(map[string]any)
	if block["type"] != "text" {
		t.Errorf("block.type = %v, want text", block["type"])
	}
	if block["text"] != "Fix the bug in auth.go" {
		t.Errorf("block.text = %v, want Fix the bug in auth.go", block["text"])
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

func TestAvailableCommandsUpdate(t *testing.T) {
	update := AvailableCommandsUpdate([]Command{
		{Name: "help", Description: "Show help"},
	})

	data := mustMarshal(t, update)
	m := mustUnmarshalMap(t, data)

	if m["sessionUpdate"] != "available_commands_update" {
		t.Errorf("sessionUpdate = %v, want available_commands_update", m["sessionUpdate"])
	}

	cmds := m["availableCommands"].([]any)
	if len(cmds) != 1 {
		t.Fatalf("availableCommands length = %d, want 1", len(cmds))
	}

	cmd := cmds[0].(map[string]any)
	if cmd["name"] != "help" {
		t.Errorf("name = %v, want help", cmd["name"])
	}
}

func TestPermissionResponseSelected(t *testing.T) {
	resp := RequestPermissionResponse{
		Outcome: PermissionOutcome{
			Selected: &SelectedOutcome{OptionId: "allow"},
		},
	}

	data := mustMarshal(t, resp)
	m := mustUnmarshalMap(t, data)

	outcome := m["outcome"].(map[string]any)
	selected := outcome["selected"].(map[string]any)
	if selected["optionId"] != "allow" {
		t.Errorf("optionId = %v, want allow", selected["optionId"])
	}

	if _, ok := outcome["cancelled"]; ok {
		t.Error("cancelled should not be present")
	}
}

func TestPermissionResponseCancelled(t *testing.T) {
	resp := RequestPermissionResponse{
		Outcome: PermissionOutcome{
			Cancelled: &struct{}{},
		},
	}

	data := mustMarshal(t, resp)
	m := mustUnmarshalMap(t, data)

	outcome := m["outcome"].(map[string]any)
	if _, ok := outcome["cancelled"]; !ok {
		t.Error("cancelled should be present")
	}
	if _, ok := outcome["selected"]; ok {
		t.Error("selected should not be present")
	}
}

func strPtr(s string) *string { return &s }
