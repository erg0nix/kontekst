package protocol

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agent"
	"github.com/erg0nix/kontekst/internal/conversation"
	"github.com/erg0nix/kontekst/internal/protocol/types"
	"github.com/erg0nix/kontekst/internal/provider"
)

type mockRunner struct {
	events  []agent.Event
	onCmd   func(agent.Command)
	onStart func(agent.RunConfig)
}

func (m *mockRunner) StartRun(cfg agent.RunConfig) (chan<- agent.Command, <-chan agent.Event, error) {
	if m.onStart != nil {
		m.onStart(cfg)
	}

	cmdCh := make(chan agent.Command, 16)
	evtCh := make(chan agent.Event, 32)

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
	agentConfig.EnsureDefaults(tmpDir)
	return agent.NewRegistry(tmpDir)
}

func setupTestPair(t *testing.T, runner agent.Runner) (*Connection, *Connection) {
	t.Helper()

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	registry := testRegistry(t)
	handler := NewHandler(runner, registry, nil)

	serverConn := handler.Serve(serverW, serverR)
	clientConn := NewConnection(nil, clientW, clientR)

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	return serverConn, clientConn
}

func initAndCreateSession(t *testing.T, client *Connection) types.SessionID {
	t.Helper()
	ctx := context.Background()

	result, err := client.Request(ctx, types.MethodInitialize, types.InitializeRequest{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	var initResp types.InitializeResponse
	json.Unmarshal(result, &initResp)
	if initResp.ProtocolVersion != 1 {
		t.Fatalf("protocol version = %d, want 1", initResp.ProtocolVersion)
	}

	result, err = client.Request(ctx, types.MethodSessionNew, types.NewSessionRequest{
		Cwd:        "/tmp",
		McpServers: []types.McpServer{},
	})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	var sessResp types.NewSessionResponse
	json.Unmarshal(result, &sessResp)
	if sessResp.SessionID == "" {
		t.Fatal("session ID is empty")
	}

	return sessResp.SessionID
}

func TestServerSimplePrompt(t *testing.T) {
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: provider.Response{Content: "Hello!"}},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	updates := make(chan json.RawMessage, 10)
	client.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == types.MethodSessionUpdate {
			updates <- params
		}
		return nil, nil
	}

	ctx := context.Background()
	result, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: sid,
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	var resp types.PromptResponse
	json.Unmarshal(result, &resp)
	if resp.StopReason != types.StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", resp.StopReason)
	}

	timeout := time.After(2 * time.Second)
	gotMessage := false
	for {
		select {
		case raw := <-updates:
			var notif types.SessionNotification
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
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: provider.Response{Content: "I'll read the file"}},
			{Type: agent.EvtToolsProposed, RunID: "run_1", Calls: []agent.ProposedToolCall{
				{CallID: "call_1", Name: "read_file", ArgumentsJSON: `{"path":"/tmp/test.go"}`},
			}},
			{Type: agent.EvtToolStarted, RunID: "run_1", CallID: "call_1"},
			{Type: agent.EvtToolCompleted, RunID: "run_1", CallID: "call_1", Output: "package main"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onCmd: func(cmd agent.Command) {
			if cmd.Type == agent.CmdApproveTool {
				approved <- cmd.CallID
			}
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == types.MethodRequestPermission {
			return types.RequestPermissionResponse{
				Outcome: types.PermissionSelected("allow"),
			}, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	result, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: sid,
		Prompt:    []types.ContentBlock{types.TextBlock("read test.go")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	var resp types.PromptResponse
	json.Unmarshal(result, &resp)
	if resp.StopReason != types.StopReasonEndTurn {
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
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1", Response: provider.Response{Content: "I'll write the file"}},
			{Type: agent.EvtToolsProposed, RunID: "run_1", Calls: []agent.ProposedToolCall{
				{CallID: "call_1", Name: "write_file", ArgumentsJSON: `{"path":"/tmp/out.go"}`},
			}},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onCmd: func(cmd agent.Command) {
			if cmd.Type == agent.CmdDenyTool {
				denied <- cmd.CallID
			}
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == types.MethodRequestPermission {
			return types.RequestPermissionResponse{
				Outcome: types.PermissionSelected("reject"),
			}, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	_, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: sid,
		Prompt:    []types.ContentBlock{types.TextBlock("write file")},
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
	eventCh := make(chan agent.Event, 32)

	runner := &mockRunnerChan{
		eventCh:        eventCh,
		cancelReceived: cancelReceived,
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	eventCh <- agent.Event{Type: agent.EvtRunStarted, RunID: "run_1"}

	ctx := context.Background()
	promptDone := make(chan struct{})
	go func() {
		client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
			SessionID: sid,
			Prompt:    []types.ContentBlock{types.TextBlock("test")},
		})
		close(promptDone)
	}()

	time.Sleep(50 * time.Millisecond)

	client.Notify(ctx, types.MethodSessionCancel, types.CancelNotification{SessionID: sid})

	select {
	case <-cancelReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel not received")
	}

	eventCh <- agent.Event{Type: agent.EvtRunCancelled, RunID: "run_1"}
	close(eventCh)

	select {
	case <-promptDone:
	case <-time.After(2 * time.Second):
		t.Fatal("prompt did not return after cancel")
	}
}

type mockRunnerChan struct {
	eventCh        chan agent.Event
	cancelReceived chan struct{}
}

func (m *mockRunnerChan) StartRun(_ agent.RunConfig) (chan<- agent.Command, <-chan agent.Event, error) {
	cmdCh := make(chan agent.Command, 16)

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
	client.Request(ctx, types.MethodInitialize, types.InitializeRequest{ProtocolVersion: 1})

	_, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: "nonexistent",
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected RPCError, got %T", err)
	}
	if rpcErr.Code != int(types.ErrNotFound) {
		t.Errorf("code = %d, want %d", rpcErr.Code, types.ErrNotFound)
	}
}

func TestServerRunFailed(t *testing.T) {
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunFailed, RunID: "run_1", Error: "model returned error"},
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	ctx := context.Background()
	_, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: sid,
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err == nil {
		t.Fatal("expected error for failed run")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected RPCError, got %T", err)
	}
	if rpcErr.Code != int(types.ErrInternalError) {
		t.Errorf("code = %d, want %d", rpcErr.Code, types.ErrInternalError)
	}
	if rpcErr.Message != "model returned error" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "model returned error")
	}
}

func TestServerConnectionCloseCancel(t *testing.T) {
	cancelReceived := make(chan struct{})
	eventCh := make(chan agent.Event, 32)

	runner := &mockRunnerChan{
		eventCh:        eventCh,
		cancelReceived: cancelReceived,
	}

	serverConn, clientConn := setupTestPair(t, runner)
	sid := initAndCreateSession(t, clientConn)

	clientConn.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	eventCh <- agent.Event{Type: agent.EvtRunStarted, RunID: "run_1"}

	promptDone := make(chan struct{})
	go func() {
		clientConn.Request(context.Background(), types.MethodSessionPrompt, types.PromptRequest{
			SessionID: sid,
			Prompt:    []types.ContentBlock{types.TextBlock("test")},
		})
		close(promptDone)
	}()

	time.Sleep(50 * time.Millisecond)

	serverConn.Close()

	select {
	case <-cancelReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel not received after connection close")
	}

	eventCh <- agent.Event{Type: agent.EvtRunCancelled, RunID: "run_1"}
	close(eventCh)

	select {
	case <-promptDone:
	case <-time.After(2 * time.Second):
		t.Fatal("prompt did not return after connection close")
	}
}

func TestIsAllowOutcome(t *testing.T) {
	options := []types.PermissionOption{
		{OptionID: "allow", Name: "Allow", Kind: types.PermissionOptionKindAllowOnce},
		{OptionID: "allow_always", Name: "Allow Always", Kind: types.PermissionOptionKindAllowAlways},
		{OptionID: "reject", Name: "Reject", Kind: types.PermissionOptionKindRejectOnce},
	}

	tests := []struct {
		outcome types.PermissionOutcome
		want    bool
	}{
		{types.PermissionSelected("allow"), true},
		{types.PermissionSelected("allow_always"), true},
		{types.PermissionSelected("reject"), false},
		{types.PermissionSelected("unknown"), false},
		{types.PermissionCancelled(), false},
	}

	for _, tt := range tests {
		got := outcomeIsAllowed(tt.outcome, options)
		if got != tt.want {
			t.Errorf("outcomeIsAllowed(%+v) = %v, want %v", tt.outcome, got, tt.want)
		}
	}
}

func TestServerNewSessionWithMeta(t *testing.T) {
	agentNameCh := make(chan string, 1)
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
	}
	runner.onStart = func(cfg agent.RunConfig) {
		agentNameCh <- cfg.AgentName
	}

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

	serverConn := handler.Serve(serverW, serverR)
	clientConn := NewConnection(nil, clientW, clientR)

	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	ctx := context.Background()
	clientConn.Request(ctx, types.MethodInitialize, types.InitializeRequest{ProtocolVersion: 1})

	result, err := clientConn.Request(ctx, types.MethodSessionNew, types.NewSessionRequest{
		Cwd:        "/project",
		McpServers: []types.McpServer{},
		Meta:       map[string]any{"agentName": "myagent"},
	})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	var resp types.NewSessionResponse
	json.Unmarshal(result, &resp)
	if resp.SessionID == "" {
		t.Fatal("session ID is empty")
	}

	clientConn.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}
	_, err = clientConn.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: resp.SessionID,
		Prompt:    []types.ContentBlock{types.TextBlock("test")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	select {
	case name := <-agentNameCh:
		if name != "myagent" {
			t.Errorf("agent name = %q, want myagent", name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("agent name not received")
	}
}

func TestServerLoadSession(t *testing.T) {
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
	}

	_, client := setupTestPair(t, runner)
	ctx := context.Background()

	client.Request(ctx, types.MethodInitialize, types.InitializeRequest{ProtocolVersion: 1})

	result, err := client.Request(ctx, types.MethodSessionLoad, types.LoadSessionRequest{
		SessionID:  "existing_sess",
		Cwd:        "/tmp",
		McpServers: []types.McpServer{},
	})
	if err != nil {
		t.Fatalf("load session failed: %v", err)
	}

	var resp types.LoadSessionResponse
	json.Unmarshal(result, &resp)
	if resp.SessionID != "existing_sess" {
		t.Errorf("sessionId = %v, want existing_sess", resp.SessionID)
	}

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	result, err = client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: "existing_sess",
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt on loaded session failed: %v", err)
	}

	var promptResp types.PromptResponse
	json.Unmarshal(result, &promptResp)
	if promptResp.StopReason != types.StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", promptResp.StopReason)
	}
}

func TestServerEventForwarding(t *testing.T) {
	tests := []struct {
		name       string
		event      agent.Event
		wantUpdate string
	}{
		{
			name:       "EvtTokenDelta",
			event:      agent.Event{Type: agent.EvtTokenDelta, RunID: "run_1", Token: "hello"},
			wantUpdate: "agent_message_chunk",
		},
		{
			name:       "EvtReasoningDelta",
			event:      agent.Event{Type: agent.EvtReasoningDelta, RunID: "run_1", Reasoning: "thinking..."},
			wantUpdate: "agent_thought_chunk",
		},
		{
			name:       "EvtToolStarted",
			event:      agent.Event{Type: agent.EvtToolStarted, RunID: "run_1", CallID: "call_1"},
			wantUpdate: "tool_call_update",
		},
		{
			name:       "EvtToolCompleted",
			event:      agent.Event{Type: agent.EvtToolCompleted, RunID: "run_1", CallID: "call_1", Output: "result"},
			wantUpdate: "tool_call_update",
		},
		{
			name:       "EvtToolFailed",
			event:      agent.Event{Type: agent.EvtToolFailed, RunID: "run_1", CallID: "call_1", Error: "oops"},
			wantUpdate: "tool_call_update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockRunner{
				events: []agent.Event{
					{Type: agent.EvtRunStarted, RunID: "run_1"},
					tt.event,
					{Type: agent.EvtRunCompleted, RunID: "run_1"},
				},
			}

			_, client := setupTestPair(t, runner)
			sid := initAndCreateSession(t, client)

			updates := make(chan json.RawMessage, 10)
			client.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
				if method == types.MethodSessionUpdate {
					updates <- params
				}
				return nil, nil
			}

			ctx := context.Background()
			_, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
				SessionID: sid,
				Prompt:    []types.ContentBlock{types.TextBlock("test")},
			})
			if err != nil {
				t.Fatalf("prompt failed: %v", err)
			}

			timeout := time.After(2 * time.Second)
			for {
				select {
				case raw := <-updates:
					var notif types.SessionNotification
					json.Unmarshal(raw, &notif)
					if m, ok := notif.Update.(map[string]any); ok {
						if m["sessionUpdate"] == tt.wantUpdate {
							return
						}
					}
				case <-timeout:
					t.Fatalf("did not receive %s update", tt.wantUpdate)
					return
				}
			}
		})
	}
}

func TestServerContextSnapshot(t *testing.T) {
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtTurnCompleted, RunID: "run_1",
				Response: provider.Response{Content: "Hi"},
				Snapshot: &conversation.Snapshot{ContextSize: 4096, HistoryTokens: 100},
			},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	contextReceived := make(chan json.RawMessage, 1)
	client.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == types.MethodKontekstContext {
			contextReceived <- params
		}
		return nil, nil
	}

	ctx := context.Background()
	_, err := client.Request(ctx, types.MethodSessionPrompt, types.PromptRequest{
		SessionID: sid,
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	select {
	case raw := <-contextReceived:
		var snapshot conversation.Snapshot
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			t.Fatalf("unmarshal snapshot: %v", err)
		}
		if snapshot.ContextSize != 4096 {
			t.Errorf("context_size = %d, want 4096", snapshot.ContextSize)
		}
		if snapshot.HistoryTokens != 100 {
			t.Errorf("history_tokens = %d, want 100", snapshot.HistoryTokens)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("context snapshot not received")
	}
}
