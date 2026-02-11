package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
)

type UpdateHandler func(SessionNotification)
type PermissionHandler func(RequestPermissionRequest) RequestPermissionResponse
type ContextSnapshotHandler func(json.RawMessage)

type ClientCallbacks struct {
	OnUpdate          UpdateHandler
	OnPermission      PermissionHandler
	OnContextSnapshot ContextSnapshotHandler
}

type Client struct {
	conn              *Connection
	OnUpdate          UpdateHandler
	OnPermission      PermissionHandler
	OnContextSnapshot ContextSnapshotHandler
}

func Dial(ctx context.Context, addr string, callbacks ClientCallbacks) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("acp: dial %s: %w", addr, err)
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

func (c *Client) Initialize(ctx context.Context, req InitializeRequest) (InitializeResponse, error) {
	result, err := c.conn.Request(ctx, MethodInitialize, req)
	if err != nil {
		return InitializeResponse{}, err
	}

	var resp InitializeResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return InitializeResponse{}, fmt.Errorf("acp: unmarshal initialize response: %w", err)
	}
	return resp, nil
}

func (c *Client) NewSession(ctx context.Context, req NewSessionRequest) (NewSessionResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionNew, req)
	if err != nil {
		return NewSessionResponse{}, err
	}

	var resp NewSessionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return NewSessionResponse{}, fmt.Errorf("acp: unmarshal session response: %w", err)
	}
	return resp, nil
}

func (c *Client) LoadSession(ctx context.Context, req LoadSessionRequest) (LoadSessionResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionLoad, req)
	if err != nil {
		return LoadSessionResponse{}, err
	}

	var resp LoadSessionResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return LoadSessionResponse{}, fmt.Errorf("acp: unmarshal load session response: %w", err)
	}
	return resp, nil
}

func (c *Client) Prompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
	result, err := c.conn.Request(ctx, MethodSessionPrompt, req)
	if err != nil {
		return PromptResponse{}, err
	}

	var resp PromptResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return PromptResponse{}, fmt.Errorf("acp: unmarshal prompt response: %w", err)
	}
	return resp, nil
}

func (c *Client) Cancel(ctx context.Context, sessionID SessionID) error {
	return c.conn.Notify(ctx, MethodSessionCancel, CancelNotification{SessionID: sessionID})
}

func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	_, err := c.conn.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: ProtocolVersion})
	if err != nil {
		return StatusResponse{}, fmt.Errorf("acp: initialize: %w", err)
	}

	statusResult, err := c.conn.Request(ctx, MethodKontekstStatus, nil)
	if err != nil {
		return StatusResponse{}, fmt.Errorf("acp: status request: %w", err)
	}

	var resp StatusResponse
	if err := json.Unmarshal(statusResult, &resp); err != nil {
		return StatusResponse{}, fmt.Errorf("acp: unmarshal status response: %w", err)
	}
	return resp, nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.conn.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: ProtocolVersion})
	if err != nil {
		return fmt.Errorf("acp: initialize: %w", err)
	}

	_, err = c.conn.Request(ctx, MethodKontekstShutdown, nil)
	if err != nil {
		return fmt.Errorf("acp: shutdown: %w", err)
	}
	return nil
}

func (c *Client) Done() <-chan struct{} {
	return c.conn.Done()
}

func (c *Client) Close() error {
	return c.conn.Close()
}
