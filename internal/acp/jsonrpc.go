package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

const maxMessageSize = 10 * 1024 * 1024

type MethodHandler func(ctx context.Context, method string, params json.RawMessage) (any, error)

type Connection struct {
	writer  io.Writer
	scanner *bufio.Scanner
	handler MethodHandler
	pending map[int]chan jsonrpcResponse
	nextID  int
	mu      sync.Mutex
	done    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
}

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonrpcResponse struct {
	Result json.RawMessage
	Error  *jsonrpcError
}

type RPCError struct {
	Code    int
	Message string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

func NewRPCError(code ErrorCode, message string) *RPCError {
	return &RPCError{Code: int(code), Message: message}
}

func NewConnection(handler MethodHandler, w io.Writer, r io.Reader) *Connection {
	c := newConnection(handler, w, r)
	go c.readLoop()
	return c
}

func newConnection(handler MethodHandler, w io.Writer, r io.Reader) *Connection {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, maxMessageSize), maxMessageSize)

	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		writer:  w,
		scanner: scanner,
		handler: handler,
		pending: make(map[int]chan jsonrpcResponse),
		done:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (c *Connection) Start() {
	go c.readLoop()
}

func (c *Connection) readLoop() {
	defer close(c.done)
	defer c.cancel()

	for c.scanner.Scan() {
		line := c.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonrpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.ID != nil && msg.Method == "" {
			c.deliverResponse(msg)
		} else if msg.ID != nil && msg.Method != "" {
			go c.handleRequest(msg)
		} else if msg.Method != "" {
			go c.handleNotification(msg)
		}
	}

	c.mu.Lock()
	for id, ch := range c.pending {
		ch <- jsonrpcResponse{Error: &jsonrpcError{Code: -32603, Message: "connection closed"}}
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

func (c *Connection) deliverResponse(msg jsonrpcMessage) {
	c.mu.Lock()
	ch, ok := c.pending[*msg.ID]
	if ok {
		delete(c.pending, *msg.ID)
	}
	c.mu.Unlock()

	if ok {
		ch <- jsonrpcResponse{Result: msg.Result, Error: msg.Error}
	}
}

func (c *Connection) handleRequest(msg jsonrpcMessage) {
	if c.handler == nil {
		resp := jsonrpcMessage{JSONRPC: "2.0", ID: msg.ID, Error: &jsonrpcError{Code: int(ErrMethodNotFound), Message: "no handler"}}
		c.writeMessage(resp)
		return
	}

	result, err := c.handler(c.ctx, msg.Method, msg.Params)

	var resp jsonrpcMessage
	resp.JSONRPC = "2.0"
	resp.ID = msg.ID

	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) {
			resp.Error = &jsonrpcError{Code: rpcErr.Code, Message: rpcErr.Message}
		} else {
			resp.Error = &jsonrpcError{Code: int(ErrInternalError), Message: err.Error()}
		}
	} else {
		data, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			resp.Error = &jsonrpcError{Code: int(ErrInternalError), Message: marshalErr.Error()}
		} else {
			resp.Result = data
		}
	}

	c.writeMessage(resp)
}

func (c *Connection) handleNotification(msg jsonrpcMessage) {
	if c.handler == nil {
		return
	}
	_, _ = c.handler(c.ctx, msg.Method, msg.Params)
}

func (c *Connection) writeMessage(msg jsonrpcMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("acp: marshal message: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	data = append(data, '\n')
	_, err = c.writer.Write(data)
	if err != nil {
		return fmt.Errorf("acp: write: %w", err)
	}
	return nil
}

func (c *Connection) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan jsonrpcResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			c.mu.Lock()
			delete(c.pending, id)
			c.mu.Unlock()
			return nil, fmt.Errorf("acp: marshal params: %w", err)
		}
	}

	msg := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  rawParams,
	}

	if err := c.writeMessage(msg); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, &RPCError{Code: resp.Error.Code, Message: resp.Error.Message}
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("acp: connection closed")
	}
}

func (c *Connection) Notify(ctx context.Context, method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("acp: marshal params: %w", err)
		}
	}

	msg := jsonrpcMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	}

	return c.writeMessage(msg)
}

func (c *Connection) Done() <-chan struct{} {
	return c.done
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) Close() error {
	c.cancel()
	if closer, ok := c.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
