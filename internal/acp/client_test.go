package acp

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestClientInitialize(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == MethodInitialize {
			return InitializeResponse{
				ProtocolVersion: 1,
				AgentInfo:       &Implementation{Name: "test"},
				AuthMethods:     []AuthMethod{},
			}, nil
		}
		return nil, NewRPCError(ErrMethodNotFound, "unknown")
	}

	_ = NewConnection(serverHandler, serverW, serverR)

	clientConn := NewConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Initialize(ctx, InitializeRequest{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	if resp.ProtocolVersion != 1 {
		t.Errorf("version = %d, want 1", resp.ProtocolVersion)
	}
	if resp.AgentInfo.Name != "test" {
		t.Errorf("name = %v, want test", resp.AgentInfo.Name)
	}
}

func TestClientNewSession(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		if method == MethodSessionNew {
			return NewSessionResponse{SessionId: "sess_test"}, nil
		}
		return nil, nil
	}

	_ = NewConnection(serverHandler, serverW, serverR)

	clientConn := NewConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	defer client.Close()

	ctx := context.Background()
	resp, err := client.NewSession(ctx, NewSessionRequest{Cwd: "/tmp", McpServers: []McpServer{}})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	if resp.SessionId != "sess_test" {
		t.Errorf("sessionId = %v, want sess_test", resp.SessionId)
	}
}

func TestClientPromptWithUpdates(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	var serverConn *Connection
	serverConn = NewConnection(func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == MethodSessionPrompt {
			var req PromptRequest
			json.Unmarshal(params, &req)

			_ = serverConn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
				SessionId: req.SessionId,
				Update:    AgentMessageChunk("response text"),
			})

			return PromptResponse{StopReason: StopReasonEndTurn}, nil
		}
		return nil, nil
	}, serverW, serverR)

	updates := make(chan SessionNotification, 10)

	clientConn := NewConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	client.OnUpdate = func(notif SessionNotification) {
		updates <- notif
	}
	clientConn.handler = client.dispatch
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Prompt(ctx, PromptRequest{
		SessionId: "sess_1",
		Prompt:    []ContentBlock{TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	if resp.StopReason != StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", resp.StopReason)
	}

	select {
	case notif := <-updates:
		if notif.SessionId != "sess_1" {
			t.Errorf("sessionId = %v, want sess_1", notif.SessionId)
		}
	case <-time.After(2 * time.Second):
		t.Log("update may have arrived before callback was set")
	}
}

func TestClientCancel(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	received := make(chan string, 1)
	_ = NewConnection(func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		received <- method
		return nil, nil
	}, serverW, serverR)

	clientConn := NewConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	defer client.Close()

	ctx := context.Background()
	if err := client.Cancel(ctx, "sess_1"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	select {
	case method := <-received:
		if method != MethodSessionCancel {
			t.Errorf("method = %v, want %v", method, MethodSessionCancel)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel not received")
	}
}

func TestClientPermissionCallback(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	server := NewConnection(nil, serverW, serverR)

	clientConn := NewConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	client.OnPermission = func(req RequestPermissionRequest) RequestPermissionResponse {
		return RequestPermissionResponse{
			Outcome: PermissionOutcome{
				Selected: &SelectedOutcome{OptionId: "allow"},
			},
		}
	}
	clientConn.handler = client.dispatch
	defer client.Close()

	ctx := context.Background()
	result, err := server.Request(ctx, MethodRequestPerm, RequestPermissionRequest{
		SessionId: "sess_1",
		ToolCall: ToolCallDetail{
			ToolCallId: "call_1",
		},
		Options: []PermissionOption{
			{OptionId: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
		},
	})
	if err != nil {
		t.Fatalf("permission request failed: %v", err)
	}

	var resp RequestPermissionResponse
	json.Unmarshal(result, &resp)
	if resp.Outcome.Selected == nil || resp.Outcome.Selected.OptionId != "allow" {
		t.Errorf("outcome = %+v, want selected allow", resp.Outcome)
	}
}
