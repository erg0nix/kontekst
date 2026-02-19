package protocol

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
)

func TestACPToolExecutorDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		caps      ClientCapabilities
		wantTools []string
	}{
		{
			name:      "no capabilities",
			caps:      ClientCapabilities{},
			wantTools: nil,
		},
		{
			name:      "fs read only",
			caps:      ClientCapabilities{Fs: &FileSystemCapability{ReadTextFile: true}},
			wantTools: []string{"read_file"},
		},
		{
			name:      "fs write only",
			caps:      ClientCapabilities{Fs: &FileSystemCapability{WriteTextFile: true}},
			wantTools: []string{"write_file"},
		},
		{
			name:      "terminal only",
			caps:      ClientCapabilities{Terminal: true},
			wantTools: []string{"run_command"},
		},
		{
			name: "all capabilities",
			caps: ClientCapabilities{
				Fs:       &FileSystemCapability{ReadTextFile: true, WriteTextFile: true},
				Terminal: true,
			},
			wantTools: []string{"read_file", "write_file", "run_command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewToolExecutor(nil, "sess_1", tt.caps)
			defs := exec.ToolDefinitions()

			if len(defs) != len(tt.wantTools) {
				t.Fatalf("got %d tools, want %d", len(defs), len(tt.wantTools))
			}

			for i, want := range tt.wantTools {
				if defs[i].Name != want {
					t.Errorf("tool[%d] = %q, want %q", i, defs[i].Name, want)
				}
			}
		})
	}
}

func TestACPToolExecutorPreview(t *testing.T) {
	exec := NewToolExecutor(nil, "sess_1", ClientCapabilities{})
	result, err := exec.Preview("read_file", nil, context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if result != "" {
		t.Errorf("Preview returned %q, want empty", result)
	}
}

func TestACPToolExecutorUnknownTool(t *testing.T) {
	exec := NewToolExecutor(nil, "sess_1", ClientCapabilities{})
	_, err := exec.Execute("nonexistent", nil, context.Background())
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestACPToolExecutorReadFile(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientConn := newConnection(func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		if method != MethodFsReadTextFile {
			t.Errorf("unexpected method: %s", method)
			return nil, nil
		}

		var req ReadTextFileRequest
		json.Unmarshal(params, &req)
		if req.Path != "/tmp/test.go" {
			t.Errorf("path = %q, want /tmp/test.go", req.Path)
		}
		if req.SessionID != "sess_1" {
			t.Errorf("sessionId = %q, want sess_1", req.SessionID)
		}

		return ReadTextFileResponse{Content: "package main"}, nil
	}, clientW, clientR)
	clientConn.Start()
	defer clientConn.Close()

	serverConn := newConnection(nil, serverW, serverR)
	serverConn.Start()
	defer serverConn.Close()

	exec := NewToolExecutor(serverConn, "sess_1", ClientCapabilities{
		Fs: &FileSystemCapability{ReadTextFile: true},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := exec.Execute("read_file", map[string]any{"path": "/tmp/test.go"}, ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "package main" {
		t.Errorf("result = %q, want %q", result, "package main")
	}
}

func TestACPToolExecutorReadFileWithLineAndLimit(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientConn := newConnection(func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		var req ReadTextFileRequest
		json.Unmarshal(params, &req)

		if req.Line == nil || *req.Line != 10 {
			t.Errorf("line = %v, want 10", req.Line)
		}
		if req.Limit == nil || *req.Limit != 5 {
			t.Errorf("limit = %v, want 5", req.Limit)
		}

		return ReadTextFileResponse{Content: "line 10\nline 11"}, nil
	}, clientW, clientR)
	clientConn.Start()
	defer clientConn.Close()

	serverConn := newConnection(nil, serverW, serverR)
	serverConn.Start()
	defer serverConn.Close()

	exec := NewToolExecutor(serverConn, "sess_1", ClientCapabilities{
		Fs: &FileSystemCapability{ReadTextFile: true},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := exec.Execute("read_file", map[string]any{
		"path":  "/tmp/test.go",
		"line":  float64(10),
		"limit": float64(5),
	}, ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "line 10\nline 11" {
		t.Errorf("result = %q, want %q", result, "line 10\nline 11")
	}
}

func TestACPToolExecutorWriteFile(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientConn := newConnection(func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		if method != MethodFsWriteTextFile {
			t.Errorf("unexpected method: %s", method)
			return nil, nil
		}

		var req WriteTextFileRequest
		json.Unmarshal(params, &req)
		if req.Path != "/tmp/out.go" {
			t.Errorf("path = %q, want /tmp/out.go", req.Path)
		}
		if req.Content != "package main" {
			t.Errorf("content = %q, want %q", req.Content, "package main")
		}

		return WriteTextFileResponse{}, nil
	}, clientW, clientR)
	clientConn.Start()
	defer clientConn.Close()

	serverConn := newConnection(nil, serverW, serverR)
	serverConn.Start()
	defer serverConn.Close()

	exec := NewToolExecutor(serverConn, "sess_1", ClientCapabilities{
		Fs: &FileSystemCapability{WriteTextFile: true},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := exec.Execute("write_file", map[string]any{
		"path":    "/tmp/out.go",
		"content": "package main",
	}, ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "wrote to /tmp/out.go" {
		t.Errorf("result = %q, want %q", result, "wrote to /tmp/out.go")
	}
}

func TestACPToolExecutorRunCommand(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientConn := newConnection(func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case MethodTerminalCreate:
			var req CreateTerminalRequest
			json.Unmarshal(params, &req)
			if req.Command != "ls" {
				t.Errorf("command = %q, want ls", req.Command)
			}
			if len(req.Args) != 1 || req.Args[0] != "-la" {
				t.Errorf("args = %v, want [-la]", req.Args)
			}
			return CreateTerminalResponse{TerminalID: "term_1"}, nil

		case MethodTerminalWait:
			var req WaitForExitRequest
			json.Unmarshal(params, &req)
			if req.TerminalID != "term_1" {
				t.Errorf("terminalId = %q, want term_1", req.TerminalID)
			}
			code := uint32(0)
			return WaitForExitResponse{ExitCode: &code}, nil

		case MethodTerminalOutput:
			return TerminalOutputResponse{Output: "file1.go\nfile2.go", Truncated: false}, nil

		case MethodTerminalRelease:
			return ReleaseTerminalResponse{}, nil

		default:
			t.Errorf("unexpected method: %s", method)
			return nil, nil
		}
	}, clientW, clientR)
	clientConn.Start()
	defer clientConn.Close()

	serverConn := newConnection(nil, serverW, serverR)
	serverConn.Start()
	defer serverConn.Close()

	exec := NewToolExecutor(serverConn, "sess_1", ClientCapabilities{Terminal: true})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := exec.Execute("run_command", map[string]any{
		"command": "ls",
		"args":    []any{"-la"},
	}, ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "file1.go\nfile2.go" {
		t.Errorf("result = %q, want %q", result, "file1.go\nfile2.go")
	}
}

func TestACPToolExecutorRunCommandNonZeroExit(t *testing.T) {
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	clientConn := newConnection(func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case MethodTerminalCreate:
			return CreateTerminalResponse{TerminalID: "term_1"}, nil
		case MethodTerminalWait:
			code := uint32(1)
			return WaitForExitResponse{ExitCode: &code}, nil
		case MethodTerminalOutput:
			return TerminalOutputResponse{Output: "error: not found", Truncated: false}, nil
		case MethodTerminalRelease:
			return ReleaseTerminalResponse{}, nil
		default:
			return nil, nil
		}
	}, clientW, clientR)
	clientConn.Start()
	defer clientConn.Close()

	serverConn := newConnection(nil, serverW, serverR)
	serverConn.Start()
	defer serverConn.Close()

	exec := NewToolExecutor(serverConn, "sess_1", ClientCapabilities{Terminal: true})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := exec.Execute("run_command", map[string]any{"command": "false"}, ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "error: not found\n[exit code: 1]" {
		t.Errorf("result = %q, want %q", result, "error: not found\n[exit code: 1]")
	}
}

func TestHasACPTools(t *testing.T) {
	tests := []struct {
		name string
		caps ClientCapabilities
		want bool
	}{
		{"empty", ClientCapabilities{}, false},
		{"fs read", ClientCapabilities{Fs: &FileSystemCapability{ReadTextFile: true}}, true},
		{"fs write", ClientCapabilities{Fs: &FileSystemCapability{WriteTextFile: true}}, true},
		{"terminal", ClientCapabilities{Terminal: true}, true},
		{"fs empty", ClientCapabilities{Fs: &FileSystemCapability{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasACPTools(tt.caps); got != tt.want {
				t.Errorf("hasACPTools() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerACPToolExecutorWired(t *testing.T) {
	var receivedTools bool
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onStart: func(cfg agent.RunConfig) {
			receivedTools = cfg.Tools != nil
		},
	}

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

	ctx := context.Background()

	_, err := clientConn.Request(ctx, MethodInitialize, InitializeRequest{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			Fs:       &FileSystemCapability{ReadTextFile: true, WriteTextFile: true},
			Terminal: true,
		},
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	result, err := clientConn.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd:        "/tmp",
		McpServers: []McpServer{},
	})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}

	var sessResp NewSessionResponse
	json.Unmarshal(result, &sessResp)

	clientConn.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	_, err = clientConn.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []ContentBlock{TextBlock("test")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	if !receivedTools {
		t.Error("RunConfig.Tools was nil, expected ToolExecutor")
	}
}

func TestServerNoCapabilitiesNoACPTools(t *testing.T) {
	var receivedTools bool
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onStart: func(cfg agent.RunConfig) {
			receivedTools = cfg.Tools != nil
		},
	}

	_, client := setupTestPair(t, runner)
	sid := initAndCreateSession(t, client)

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	ctx := context.Background()
	_, err := client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionID: sid,
		Prompt:    []ContentBlock{TextBlock("test")},
	})
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}

	if receivedTools {
		t.Error("RunConfig.Tools was set, expected nil for no capabilities")
	}
}

func TestFormatCommandOutput(t *testing.T) {
	tests := []struct {
		name string
		out  TerminalOutputResponse
		exit WaitForExitResponse
		want string
	}{
		{
			name: "simple output",
			out:  TerminalOutputResponse{Output: "hello"},
			exit: WaitForExitResponse{ExitCode: uintPtr(0)},
			want: "hello",
		},
		{
			name: "non-zero exit",
			out:  TerminalOutputResponse{Output: "fail"},
			exit: WaitForExitResponse{ExitCode: uintPtr(1)},
			want: "fail\n[exit code: 1]",
		},
		{
			name: "signal",
			out:  TerminalOutputResponse{Output: ""},
			exit: WaitForExitResponse{Signal: strPtr("SIGKILL")},
			want: "\n[signal: SIGKILL]",
		},
		{
			name: "truncated",
			out:  TerminalOutputResponse{Output: "data", Truncated: true},
			exit: WaitForExitResponse{ExitCode: uintPtr(0)},
			want: "data\n[output truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommandOutput(tt.out, tt.exit)
			if got != tt.want {
				t.Errorf("formatCommandOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func uintPtr(v uint32) *uint32 { return &v }
func strPtr(v string) *string  { return &v }
