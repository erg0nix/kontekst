package acp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/core"
)

type mockRunner struct {
	events []agent.AgentEvent
	onCmd  func(agent.AgentCommand)
}

func (m *mockRunner) StartRun(_ agent.RunConfig) (chan<- agent.AgentCommand, <-chan agent.AgentEvent, error) {
	cmdCh := make(chan agent.AgentCommand, 16)
	evtCh := make(chan agent.AgentEvent, 32)

	go func() {
		for _, e := range m.events {
			evtCh <- e
		}
		close(evtCh)
	}()

	if m.onCmd != nil {
		go func() {
			for cmd := range cmdCh {
				m.onCmd(cmd)
			}
		}()
	}

	return cmdCh, evtCh, nil
}

func testRegistry(t *testing.T) *agent.Registry {
	t.Helper()
	tmpDir := t.TempDir()
	agentConfig.EnsureDefault(tmpDir)
	return agent.NewRegistry(tmpDir)
}

func setupTestPair(t *testing.T, runner agent.Runner) (*Connection, *Connection) {
	t.Helper()

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	registry := testRegistry(t)
	handler := NewHandler(runner, registry, nil)

	serverConn := NewConnection(handler.Dispatch, serverW, serverR)
	handler.SetConnection(serverConn)

	clientConn := NewConnection(nil, clientW, clientR)

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	return serverConn, clientConn
}

func initAndCreateSession(t *testing.T, client *Connection) SessionId {
	t.Helper()
	ctx := context.Background()

	result, err := client.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	var initResp InitializeResponse
	json.Unmarshal(result, &initResp)
	if initResp.ProtocolVersion != 1 {
		t.Fatalf("protocol version = %d, want 1", initResp.ProtocolVersion)
	}

	result, err = client.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd:        "/tmp",
		McpServers: []McpServer{},
	})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	var sessResp NewSessionResponse
	json.Unmarshal(result, &sessResp)
	if sessResp.SessionId == "" {
		t.Fatal("session ID is empty")
	}

	return sessResp.SessionId
}

func TestServerSimplePrompt(t *testing.T) {
	runner := &mockRunner{
		events: []agent.AgentEvent{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: core.ChatResponse{Content: "Hello!"}},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	updates := make(chan json.RawMessage, 10)
	client.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == MethodSessionUpdate {
			updates <- params
		}
		return nil, nil
	}

	ctx := context.Background()
	result, err := client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionId: sid,
		Prompt:    []ContentBlock{TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	var resp PromptResponse
	json.Unmarshal(result, &resp)
	if resp.StopReason != StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", resp.StopReason)
	}

	timeout := time.After(2 * time.Second)
	gotMessage := false
	for {
		select {
		case raw := <-updates:
			var notif SessionNotification
			json.Unmarshal(raw, &notif)
			if m, ok := notif.Update.(map[string]any); ok {
				if m["sessionUpdate"] == "agent_message_chunk" {
					gotMessage = true
				}
			}
		case <-timeout:
			if !gotMessage {
				t.Error("did not receive agent_message_chunk update")
			}
			return
		default:
			if gotMessage {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestServerToolApproval(t *testing.T) {
	approved := make(chan string, 1)
	runner := &mockRunner{
		events: []agent.AgentEvent{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: core.ChatResponse{Content: "I'll read the file"}},
			{Type: agent.EvtToolsProposed, RunID: "run_1", Calls: []agent.ProposedToolCall{
				{CallID: "call_1", Name: "read_file", ArgumentsJSON: `{"path":"/tmp/test.go"}`},
			}},
			{Type: agent.EvtToolStarted, RunID: "run_1", CallID: "call_1"},
			{Type: agent.EvtToolCompleted, RunID: "run_1", CallID: "call_1", Output: "package main"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onCmd: func(cmd agent.AgentCommand) {
			if cmd.Type == agent.CmdApproveTool {
				approved <- cmd.CallID
			}
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == MethodRequestPerm {
			return RequestPermissionResponse{
				Outcome: PermissionOutcome{
					Selected: &SelectedOutcome{OptionId: "allow"},
				},
			}, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	result, err := client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionId: sid,
		Prompt:    []ContentBlock{TextBlock("read test.go")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	var resp PromptResponse
	json.Unmarshal(result, &resp)
	if resp.StopReason != StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", resp.StopReason)
	}

	select {
	case callID := <-approved:
		if callID != "call_1" {
			t.Errorf("approved callID = %v, want call_1", callID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tool approval not received")
	}
}

func TestServerToolDenied(t *testing.T) {
	denied := make(chan string, 1)
	runner := &mockRunner{
		events: []agent.AgentEvent{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: core.ChatResponse{Content: "I'll write the file"}},
			{Type: agent.EvtToolsProposed, RunID: "run_1", Calls: []agent.ProposedToolCall{
				{CallID: "call_1", Name: "write_file", ArgumentsJSON: `{"path":"/tmp/out.go"}`},
			}},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onCmd: func(cmd agent.AgentCommand) {
			if cmd.Type == agent.CmdDenyTool {
				denied <- cmd.CallID
			}
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == MethodRequestPerm {
			return RequestPermissionResponse{
				Outcome: PermissionOutcome{
					Selected: &SelectedOutcome{OptionId: "reject"},
				},
			}, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	_, err := client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionId: sid,
		Prompt:    []ContentBlock{TextBlock("write file")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	select {
	case callID := <-denied:
		if callID != "call_1" {
			t.Errorf("denied callID = %v, want call_1", callID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tool denial not received")
	}
}

func TestServerCancel(t *testing.T) {
	cancelReceived := make(chan struct{})
	eventCh := make(chan agent.AgentEvent, 32)

	runner := &mockRunnerChan{
		eventCh:        eventCh,
		cancelReceived: cancelReceived,
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	eventCh <- agent.AgentEvent{Type: agent.EvtRunStarted, RunID: "run_1"}

	ctx := context.Background()
	promptDone := make(chan struct{})
	go func() {
		client.Request(ctx, MethodSessionPrompt, PromptRequest{
			SessionId: sid,
			Prompt:    []ContentBlock{TextBlock("test")},
		})
		close(promptDone)
	}()

	time.Sleep(50 * time.Millisecond)

	client.Notify(ctx, MethodSessionCancel, CancelNotification{SessionId: sid})

	select {
	case <-cancelReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel not received")
	}

	eventCh <- agent.AgentEvent{Type: agent.EvtRunCancelled, RunID: "run_1"}
	close(eventCh)

	select {
	case <-promptDone:
	case <-time.After(2 * time.Second):
		t.Fatal("prompt did not return after cancel")
	}
}

type mockRunnerChan struct {
	eventCh        chan agent.AgentEvent
	cancelReceived chan struct{}
}

func (m *mockRunnerChan) StartRun(_ agent.RunConfig) (chan<- agent.AgentCommand, <-chan agent.AgentEvent, error) {
	cmdCh := make(chan agent.AgentCommand, 16)

	go func() {
		for cmd := range cmdCh {
			if cmd.Type == agent.CmdCancel {
				close(m.cancelReceived)
			}
		}
	}()

	return cmdCh, m.eventCh, nil
}

func TestParseSkillInvocation(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs string
	}{
		{"/help", "help", ""},
		{"/commit -m fix bug", "commit", "-m fix bug"},
		{"/review", "review", ""},
	}

	for _, tt := range tests {
		name, args := parseSkillInvocation(tt.input)
		if name != tt.wantName || args != tt.wantArgs {
			t.Errorf("parseSkillInvocation(%q) = (%q, %q), want (%q, %q)", tt.input, name, args, tt.wantName, tt.wantArgs)
		}
	}
}

func TestServerSessionNotFound(t *testing.T) {
	runner := &mockRunner{}
	_, client := setupTestPair(t, runner)

	ctx := context.Background()
	client.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: 1})

	_, err := client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionId: "nonexistent",
		Prompt:    []ContentBlock{TextBlock("hello")},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected RPCError, got %T", err)
	}
	if rpcErr.Code != int(ErrNotFound) {
		t.Errorf("code = %d, want %d", rpcErr.Code, ErrNotFound)
	}
}

func TestServerNewSessionWithMeta(t *testing.T) {
	runner := &mockRunner{}

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents", "myagent")
	os.MkdirAll(agentDir, 0o755)
	os.WriteFile(filepath.Join(agentDir, "config.toml"), []byte(`name = "My Agent"
context_size = 4096
[provider]
endpoint = "http://localhost:8080"
model = "test"
`), 0o644)

	registry := agent.NewRegistry(tmpDir)
	handler := NewHandler(runner, registry, nil)

	serverConn := NewConnection(handler.Dispatch, serverW, serverR)
	handler.SetConnection(serverConn)

	clientConn := NewConnection(nil, clientW, clientR)

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	ctx := context.Background()
	clientConn.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: 1})

	result, err := clientConn.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd:        "/project",
		McpServers: []McpServer{},
		Meta:       map[string]any{"agentName": "myagent"},
	})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	var resp NewSessionResponse
	json.Unmarshal(result, &resp)
	if resp.SessionId == "" {
		t.Fatal("session ID is empty")
	}
}
