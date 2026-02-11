package acp

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

func TestRequestResponseRoundTrip(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == "echo" {
			var m map[string]any
			json.Unmarshal(params, &m)
			return m, nil
		}
		return nil, NewRPCError(ErrMethodNotFound, "unknown method")
	}

	server := NewConnection(serverHandler, serverW, serverR)
	defer server.Close()

	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx := context.Background()
	result, err := client.Request(ctx, "echo", map[string]any{"msg": "hello"})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if m["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", m["msg"])
	}
}

func TestNotificationDelivery(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	received := make(chan string, 1)
	serverHandler := func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		received <- method
		return nil, nil
	}

	server := NewConnection(serverHandler, serverW, serverR)
	defer server.Close()

	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx := context.Background()
	if err := client.Notify(ctx, "session/update", map[string]any{"data": "test"}); err != nil {
		t.Fatalf("notify failed: %v", err)
	}

	select {
	case method := <-received:
		if method != "session/update" {
			t.Errorf("method = %v, want session/update", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("notification not received")
	}
}

func TestIncomingRequestDispatch(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientHandler := func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method == "session/request_permission" {
			return map[string]any{"outcome": map[string]any{"selected": map[string]any{"optionId": "allow"}}}, nil
		}
		return nil, nil
	}

	server := NewConnection(nil, serverW, serverR)
	defer server.Close()

	_ = NewConnection(clientHandler, clientW, clientR)

	ctx := context.Background()
	result, err := server.Request(ctx, "session/request_permission", map[string]any{"sessionId": "s1"})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var m map[string]any
	json.Unmarshal(result, &m)
	outcome := m["outcome"].(map[string]any)
	selected := outcome["selected"].(map[string]any)
	if selected["optionId"] != "allow" {
		t.Errorf("optionId = %v, want allow", selected["optionId"])
	}
}

func TestContextCancellation(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(ctx context.Context, _ string, _ json.RawMessage) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	_ = NewConnection(serverHandler, serverW, serverR)

	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := client.Request(ctx, "slow", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestEOFClosesDone(t *testing.T) {
	r, w := io.Pipe()

	conn := NewConnection(nil, io.Discard, r)

	w.Close()

	select {
	case <-conn.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done() not closed after EOF")
	}
}

func TestConcurrentRequests(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(_ context.Context, method string, params json.RawMessage) (any, error) {
		var m map[string]any
		json.Unmarshal(params, &m)
		return map[string]any{"echo": m["n"]}, nil
	}

	server := NewConnection(serverHandler, serverW, serverR)
	defer server.Close()

	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			result, err := client.Request(ctx, "echo", map[string]any{"n": n})
			if err != nil {
				t.Errorf("request %d failed: %v", n, err)
				return
			}

			var m map[string]any
			json.Unmarshal(result, &m)
			if int(m["echo"].(float64)) != n {
				t.Errorf("echo = %v, want %d", m["echo"], n)
			}
		}(i)
	}

	wg.Wait()
}

func TestErrorResponse(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	serverHandler := func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, NewRPCError(ErrMethodNotFound, "no such method")
	}

	server := NewConnection(serverHandler, serverW, serverR)
	defer server.Close()

	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx := context.Background()
	_, err := client.Request(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected RPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != int(ErrMethodNotFound) {
		t.Errorf("code = %d, want %d", rpcErr.Code, ErrMethodNotFound)
	}
	if rpcErr.Message != "no such method" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "no such method")
	}
}

func TestNonBlockingDispatch(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	promptStarted := make(chan struct{})
	cancelReceived := make(chan struct{})

	serverHandler := func(_ context.Context, method string, _ json.RawMessage) (any, error) {
		switch method {
		case "session/prompt":
			close(promptStarted)
			<-cancelReceived
			return map[string]any{"stopReason": "cancelled"}, nil
		case "session/cancel":
			close(cancelReceived)
			return nil, nil
		}
		return nil, nil
	}

	_ = NewConnection(serverHandler, serverW, serverR)
	client := NewConnection(nil, clientW, clientR)
	defer client.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := client.Request(ctx, "session/prompt", map[string]any{"sessionId": "s1"})
		if err != nil {
			t.Errorf("prompt request failed: %v", err)
			return
		}
		var m map[string]any
		json.Unmarshal(result, &m)
		if m["stopReason"] != "cancelled" {
			t.Errorf("stopReason = %v, want cancelled", m["stopReason"])
		}
	}()

	<-promptStarted

	if err := client.Notify(ctx, "session/cancel", map[string]any{"sessionId": "s1"}); err != nil {
		t.Fatalf("cancel notify failed: %v", err)
	}

	wg.Wait()
}
