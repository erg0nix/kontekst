package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/conversation"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/session"
)

type mockContextWindow struct {
	mu       sync.Mutex
	messages []core.Message
}

func (m *mockContextWindow) SystemContent() string                    { return "" }
func (m *mockContextWindow) StartRun(conversation.BudgetParams) error { return nil }
func (m *mockContextWindow) CompleteRun()                             {}
func (m *mockContextWindow) SetActiveSkill(*core.SkillMetadata)       {}
func (m *mockContextWindow) ActiveSkill() *core.SkillMetadata         { return nil }
func (m *mockContextWindow) SetAgentSystemPrompt(string)              {}
func (m *mockContextWindow) Snapshot() conversation.Snapshot          { return conversation.Snapshot{} }

func (m *mockContextWindow) BuildContext() ([]core.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]core.Message, len(m.messages))
	copy(out, m.messages)
	return out, nil
}

func (m *mockContextWindow) AddMessage(msg core.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

type mockCtxService struct {
	window conversation.Window
}

func (m *mockCtxService) NewWindow(core.SessionID) (conversation.Window, error) {
	return m.window, nil
}

type mockSessions struct{}

func (m *mockSessions) Create() (core.SessionID, string, error)        { return "test", "/tmp", nil }
func (m *mockSessions) Ensure(core.SessionID) (string, error)          { return "/tmp", nil }
func (m *mockSessions) GetDefaultAgent(core.SessionID) (string, error) { return "", nil }
func (m *mockSessions) SetDefaultAgent(core.SessionID, string) error   { return nil }
func (m *mockSessions) List() ([]session.Info, error)                  { return nil, nil }
func (m *mockSessions) Get(core.SessionID) (session.Info, error) {
	return session.Info{}, nil
}
func (m *mockSessions) Delete(core.SessionID) error { return nil }

func testRegistryWithEndpoint(t *testing.T, endpoint string) *agent.Registry {
	t.Helper()
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents", "default")
	os.MkdirAll(agentDir, 0o755)
	os.WriteFile(filepath.Join(agentDir, "config.toml"), []byte(fmt.Sprintf(`name = "Test"
context_size = 4096
[provider]
endpoint = %q
model = "test"
`, endpoint)), 0o644)
	return agent.NewRegistry(tmpDir)
}

func TestIntegrationACPReadFile(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			json.NewEncoder(w).Encode(map[string]any{"count": 10})
			return
		}

		mu.Lock()
		call := callCount
		callCount++
		mu.Unlock()

		if call == 0 {
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{
					"message": map[string]any{
						"content": "Reading file.",
						"tool_calls": []any{map[string]any{
							"id":   "call_1",
							"type": "function",
							"function": map[string]any{
								"name":      "read_file",
								"arguments": `{"path":"/tmp/hello.txt"}`,
							},
						}},
					},
				}},
				"usage": map[string]any{"total_tokens": 70},
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": "The file says hello world"},
			}},
			"usage": map[string]any{"total_tokens": 90},
		})
	}))
	defer llm.Close()

	runner := &agent.DefaultRunner{
		Context:  &mockCtxService{window: &mockContextWindow{}},
		Sessions: &mockSessions{},
	}

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	registry := testRegistryWithEndpoint(t, llm.URL)
	handler := NewHandler(runner, registry, nil)

	serverConn := handler.Serve(serverW, serverR)
	clientConn := NewConnection(nil, clientW, clientR)
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	gotReadFile := make(chan struct{}, 1)

	clientConn.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case MethodRequestPermission:
			return RequestPermissionResponse{Outcome: PermissionSelected("allow")}, nil

		case MethodFsReadTextFile:
			var req ReadTextFileRequest
			json.Unmarshal(params, &req)
			if req.Path != "/tmp/hello.txt" {
				t.Errorf("read path = %q, want /tmp/hello.txt", req.Path)
			}
			if req.SessionID == "" {
				t.Error("sessionId is empty in reverse request")
			}
			select {
			case gotReadFile <- struct{}{}:
			default:
			}
			return ReadTextFileResponse{Content: "hello world"}, nil
		}

		return nil, nil
	}

	ctx := context.Background()
	clientConn.Request(ctx, MethodInitialize, InitializeRequest{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			Fs: &FileSystemCapability{ReadTextFile: true},
		},
	})

	result, _ := clientConn.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd: "/tmp", McpServers: []McpServer{},
	})
	var sessResp NewSessionResponse
	json.Unmarshal(result, &sessResp)

	promptResult, err := clientConn.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []ContentBlock{TextBlock("read /tmp/hello.txt")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	var promptResp PromptResponse
	json.Unmarshal(promptResult, &promptResp)
	if promptResp.StopReason != StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", promptResp.StopReason)
	}

	select {
	case <-gotReadFile:
	case <-time.After(2 * time.Second):
		t.Fatal("never received fs/read_text_file reverse request")
	}
}

func TestIntegrationACPRunCommand(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			json.NewEncoder(w).Encode(map[string]any{"count": 10})
			return
		}

		mu.Lock()
		call := callCount
		callCount++
		mu.Unlock()

		if call == 0 {
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{
					"message": map[string]any{
						"content": "Running command.",
						"tool_calls": []any{map[string]any{
							"id":   "call_1",
							"type": "function",
							"function": map[string]any{
								"name":      "run_command",
								"arguments": `{"command":"echo","args":["hello"]}`,
							},
						}},
					},
				}},
				"usage": map[string]any{"total_tokens": 50},
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": "Done"},
			}},
			"usage": map[string]any{"total_tokens": 40},
		})
	}))
	defer llm.Close()

	runner := &agent.DefaultRunner{
		Context:  &mockCtxService{window: &mockContextWindow{}},
		Sessions: &mockSessions{},
	}

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	registry := testRegistryWithEndpoint(t, llm.URL)
	handler := NewHandler(runner, registry, nil)

	serverConn := handler.Serve(serverW, serverR)
	clientConn := NewConnection(nil, clientW, clientR)
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	seenMethods := &sync.Map{}

	clientConn.handler = func(_ context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case MethodRequestPermission:
			return RequestPermissionResponse{Outcome: PermissionSelected("allow")}, nil

		case MethodTerminalCreate:
			seenMethods.Store(method, true)
			var req CreateTerminalRequest
			json.Unmarshal(params, &req)
			if req.Command != "echo" {
				t.Errorf("command = %q, want echo", req.Command)
			}
			return CreateTerminalResponse{TerminalID: "term_1"}, nil

		case MethodTerminalWait:
			seenMethods.Store(method, true)
			code := uint32(0)
			return WaitForExitResponse{ExitCode: &code}, nil

		case MethodTerminalOutput:
			seenMethods.Store(method, true)
			return TerminalOutputResponse{Output: "hello\n", Truncated: false}, nil

		case MethodTerminalRelease:
			seenMethods.Store(method, true)
			return ReleaseTerminalResponse{}, nil
		}

		return nil, nil
	}

	ctx := context.Background()
	clientConn.Request(ctx, MethodInitialize, InitializeRequest{
		ProtocolVersion:    1,
		ClientCapabilities: ClientCapabilities{Terminal: true},
	})

	result, _ := clientConn.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd: "/tmp", McpServers: []McpServer{},
	})
	var sessResp NewSessionResponse
	json.Unmarshal(result, &sessResp)

	promptResult, err := clientConn.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []ContentBlock{TextBlock("run echo hello")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	var promptResp PromptResponse
	json.Unmarshal(promptResult, &promptResp)
	if promptResp.StopReason != StopReasonEndTurn {
		t.Errorf("stopReason = %v, want end_turn", promptResp.StopReason)
	}

	for _, method := range []string{MethodTerminalCreate, MethodTerminalWait, MethodTerminalOutput, MethodTerminalRelease} {
		if _, ok := seenMethods.Load(method); !ok {
			t.Errorf("never received %s reverse request", method)
		}
	}
}

func TestIntegrationNoCapabilitiesUsesKontekstTools(t *testing.T) {
	var receivedConfig agent.RunConfig
	runner := &mockRunner{
		events: []agent.Event{
			{Type: agent.EvtRunStarted, RunID: "run_1"},
			{Type: agent.EvtRunCompleted, RunID: "run_1"},
		},
		onStart: func(cfg agent.RunConfig) {
			receivedConfig = cfg
		},
	}

	_, client := setupTestPair(t, runner)
	ctx := context.Background()

	client.Request(ctx, MethodInitialize, InitializeRequest{ProtocolVersion: 1})

	result, _ := client.Request(ctx, MethodSessionNew, NewSessionRequest{
		Cwd: "/tmp", McpServers: []McpServer{},
	})
	var sessResp NewSessionResponse
	json.Unmarshal(result, &sessResp)

	client.handler = func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, nil
	}

	client.Request(ctx, MethodSessionPrompt, PromptRequest{
		SessionID: sessResp.SessionID,
		Prompt:    []ContentBlock{TextBlock("hello")},
	})

	if receivedConfig.Tools != nil {
		t.Error("expected nil Tools (kontekst mode), got ToolExecutor")
	}
}
