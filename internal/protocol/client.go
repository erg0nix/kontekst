package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
)

// UpdateHandler is a callback invoked when the server sends a session update notification.
type UpdateHandler func(SessionNotification)

// PermissionHandler is a callback invoked when the server requests tool execution approval.
type PermissionHandler func(RequestPermissionRequest) RequestPermissionResponse

// ContextSnapshotHandler is a callback invoked when the server sends a context snapshot.
type ContextSnapshotHandler func(json.RawMessage)

// ClientCallbacks holds the callback functions for handling server-initiated messages.
type ClientCallbacks struct {
	OnUpdate          UpdateHandler
	OnPermission      PermissionHandler
	OnContextSnapshot ContextSnapshotHandler
}

// Client is an ACP client that communicates with the agent server over a JSON-RPC connection.
type Client struct {
	conn              *Connection
	OnUpdate          UpdateHandler
	OnPermission      PermissionHandler
	OnContextSnapshot ContextSnapshotHandler
}

// Dial connects to an ACP server at the given address and returns a Client.
func Dial(ctx context.Context, addr string, callbacks ClientCallbacks) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("protocol: dial %s: %w", addr, err)
	}

	client := &Client{
		OnUpdate:          callbacks.OnUpdate,
		OnPermission:      callbacks.OnPermission,
		OnContextSnapshot: callbacks.OnContextSnapshot,
	}
	client.conn = NewConnection(client.dispatch, conn, conn)

	go func() {
		<-client.conn.Done()
		conn.Close()
	}()

	return client, nil
}

// NewClient creates a Client from an existing Connection and registers its dispatch handler.
func NewClient(conn *Connection) *Client {
	c := &Client{conn: conn}
	conn.handler = c.dispatch
	return c
}

func (c *Client) dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case MethodSessionUpdate:
		if c.OnUpdate != nil {
			var notif SessionNotification
			if err := json.Unmarshal(params, &notif); err == nil {
				c.OnUpdate(notif)
			}
		}
		return nil, nil

	case MethodRequestPermission:
		if c.OnPermission != nil {
			var req RequestPermissionRequest
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, NewRPCError(ErrInvalidParams, err.Error())
			}
			return c.OnPermission(req), nil
		}
		return RequestPermissionResponse{
			Outcome: PermissionCancelled(),
		}, nil

	case MethodKontekstContext:
		if c.OnContextSnapshot != nil {
			c.OnContextSnapshot(params)
		}
		return nil, nil
	}

	return nil, nil
}

// Initialize performs the ACP protocol handshake with the server.
func (c *Client) Initialize(ctx context.Context, req InitializeRequest) (InitializeResponse, error) {
	result, err := c.conn.Request(ctx, MethodInitialize, req)
	if err != nil {
		return InitializeResponse{}, err
	}

	var resp InitializeResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return InitializeResponse{}, fmt.Errorf("protocol: unmarshal initialize response: %w", err)
	}
	return resp, nil
}

// NewSession creates a new session on the server.
func (c *Client) NewSession(ctx context.Context, req NewSessionRequest) (NewSessionResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionNew, req)
	if err != nil {
		return NewSessionResponse{}, err
	}

	var resp NewSessionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return NewSessionResponse{}, fmt.Errorf("protocol: unmarshal session response: %w", err)
	}
	return resp, nil
}

// LoadSession resumes an existing session on the server.
func (c *Client) LoadSession(ctx context.Context, req LoadSessionRequest) (LoadSessionResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionLoad, req)
	if err != nil {
		return LoadSessionResponse{}, err
	}

	var resp LoadSessionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return LoadSessionResponse{}, fmt.Errorf("protocol: unmarshal load session response: %w", err)
	}
	return resp, nil
}

// Prompt sends a user prompt to the server and blocks until the agent completes its response.
func (c *Client) Prompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionPrompt, req)
	if err != nil {
		return PromptResponse{}, err
	}

	var resp PromptResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return PromptResponse{}, fmt.Errorf("protocol: unmarshal prompt response: %w", err)
	}
	return resp, nil
}

// Cancel sends a cancellation notification for the active prompt in the given session.
func (c *Client) Cancel(ctx context.Context, sessionID SessionID) error {
	return c.conn.Notify(ctx, MethodSessionCancel, CancelNotification{SessionID: sessionID})
}

// Status queries the server for its current status after performing a handshake.
func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	_, err := c.conn.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: ProtocolVersion})
	if err != nil {
		return StatusResponse{}, fmt.Errorf("protocol: initialize: %w", err)
	}

	statusResult, err := c.conn.Request(ctx, MethodKontekstStatus, nil)
	if err != nil {
		return StatusResponse{}, fmt.Errorf("protocol: status request: %w", err)
	}

	var resp StatusResponse
	if err := json.Unmarshal(statusResult, &resp); err != nil {
		return StatusResponse{}, fmt.Errorf("protocol: unmarshal status response: %w", err)
	}
	return resp, nil
}

// Shutdown requests the server to shut down gracefully after performing a handshake.
func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.conn.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: ProtocolVersion})
	if err != nil {
		return fmt.Errorf("protocol: initialize: %w", err)
	}

	_, err = c.conn.Request(ctx, MethodKontekstShutdown, nil)
	if err != nil {
		return fmt.Errorf("protocol: shutdown: %w", err)
	}
	return nil
}

// Done returns a channel that is closed when the client's connection is closed.
func (c *Client) Done() <-chan struct{} {
	return c.conn.Done()
}

// Close shuts down the client's underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
