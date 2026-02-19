package protocol

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestClientPromptWithUpdates(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	var serverConn *Connection
	serverConn = NewConnection(func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == MethodSessionPrompt {
			var req PromptRequest
			json.Unmarshal(params, &req)

			_ = serverConn.Notify(context.Background(), MethodSessionUpdate, SessionNotification{
				SessionID: req.SessionID,
				Update:    AgentMessageChunk("response text"),
			})

			return PromptResponse{StopReason: StopReasonEndTurn}, nil
		}
		return nil, nil
	}, serverW, serverR)

	updates := make(chan SessionNotification, 10)

	clientConn := newConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	client.OnUpdate = func(notif SessionNotification) {
		updates <- notif
	}
	clientConn.Start()
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Prompt(ctx, PromptRequest{
		SessionID: "sess_1",
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
		if notif.SessionID != "sess_1" {
			t.Errorf("sessionId = %v, want sess_1", notif.SessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("update not received")
	}
}

func TestClientPermissionCallback(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	server := NewConnection(nil, serverW, serverR)

	clientConn := newConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	client.OnPermission = func(req RequestPermissionRequest) RequestPermissionResponse {
		return RequestPermissionResponse{
			Outcome: PermissionSelected("allow"),
		}
	}
	clientConn.Start()
	defer client.Close()

	ctx := context.Background()
	result, err := server.Request(ctx, MethodRequestPermission, RequestPermissionRequest{
		SessionID: "sess_1",
		ToolCall: ToolCallDetail{
			ToolCallID: "call_1",
		},
		Options: []PermissionOption{
			{OptionID: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
		},
	})
	if err != nil {
		t.Fatalf("permission request failed: %v", err)
	}

	var resp RequestPermissionResponse
	json.Unmarshal(result, &resp)
	if resp.Outcome.Outcome != "selected" || resp.Outcome.OptionID != "allow" {
		t.Errorf("outcome = %+v, want selected allow", resp.Outcome)
	}
}
