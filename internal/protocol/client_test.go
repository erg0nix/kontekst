package protocol

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/protocol/types"
)

func TestClientPromptWithUpdates(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	var serverConn *Connection
	serverConn = NewConnection(func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == types.MethodSessionPrompt {
			var req types.PromptRequest
			json.Unmarshal(params, &req)

			_ = serverConn.Notify(context.Background(), types.MethodSessionUpdate, types.SessionNotification{
				SessionID: req.SessionID,
				Update:    types.AgentMessageChunk("response text"),
			})

			return types.PromptResponse{StopReason: types.StopReasonEndTurn}, nil
		}
		return nil, nil
	}, serverW, serverR)

	updates := make(chan types.SessionNotification, 10)

	clientConn := newConnection(nil, clientW, clientR)
	client := NewClient(clientConn)
	client.OnUpdate = func(notif types.SessionNotification) {
		updates <- notif
	}
	clientConn.Start()
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Prompt(ctx, types.PromptRequest{
		SessionID: "sess_1",
		Prompt:    []types.ContentBlock{types.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	if resp.StopReason != types.StopReasonEndTurn {
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
	client.OnPermission = func(req types.RequestPermissionRequest) types.RequestPermissionResponse {
		return types.RequestPermissionResponse{
			Outcome: types.PermissionSelected("allow"),
		}
	}
	clientConn.Start()
	defer client.Close()

	ctx := context.Background()
	result, err := server.Request(ctx, types.MethodRequestPermission, types.RequestPermissionRequest{
		SessionID: "sess_1",
		ToolCall: types.ToolCallDetail{
			ToolCallID: "call_1",
		},
		Options: []types.PermissionOption{
			{OptionID: "allow", Name: "Allow", Kind: types.PermissionOptionKindAllowOnce},
		},
	})
	if err != nil {
		t.Fatalf("permission request failed: %v", err)
	}

	var resp types.RequestPermissionResponse
	json.Unmarshal(result, &resp)
	if resp.Outcome.Outcome != "selected" || resp.Outcome.OptionID != "allow" {
		t.Errorf("outcome = %+v, want selected allow", resp.Outcome)
	}
}
