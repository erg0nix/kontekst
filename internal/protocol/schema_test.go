package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/erg0nix/kontekst/internal/protocol/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	schemaCompilerOnce sync.Once
	schemaCompiler     *jsonschema.Compiler
	schemaCompilerErr  error
)

func setupSchemaCompiler() {
	data, err := os.ReadFile("schema.json")
	if err != nil {
		schemaCompilerErr = fmt.Errorf("read schema: %w", err)
		return
	}

	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		schemaCompilerErr = fmt.Errorf("parse schema: %w", err)
		return
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", doc); err != nil {
		schemaCompilerErr = fmt.Errorf("add resource: %w", err)
		return
	}
	schemaCompiler = c
}

func assertSchemaValid(t *testing.T, defName string, v any) {
	t.Helper()

	schemaCompilerOnce.Do(setupSchemaCompiler)
	if schemaCompilerErr != nil {
		t.Fatalf("schema setup: %v", schemaCompilerErr)
	}

	sch, err := schemaCompiler.Compile("schema.json#/$defs/" + defName)
	if err != nil {
		t.Fatalf("compile %s: %v", defName, err)
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}

	if err := sch.Validate(inst); err != nil {
		t.Errorf("%s validation failed:\n  JSON: %s\n  Error: %v", defName, data, err)
	}
}

func TestSchemaResponses(t *testing.T) {
	t.Run("InitializeResponse", func(t *testing.T) {
		assertSchemaValid(t, "InitializeResponse", types.InitializeResponse{
			ProtocolVersion: 1,
			AgentCapabilities: types.AgentCapabilities{
				LoadSession: true,
			},
			AgentInfo: &types.Implementation{
				Name:    "kontekst",
				Title:   "Kontekst",
				Version: "0.1.0",
			},
			AuthMethods: []types.AuthMethod{},
		})
	})

	t.Run("NewSessionResponse", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionResponse", types.NewSessionResponse{
			SessionID: "sess_123",
		})
	})

	t.Run("LoadSessionResponse", func(t *testing.T) {
		assertSchemaValid(t, "LoadSessionResponse", types.LoadSessionResponse{
			SessionID: "sess_123",
		})
	})

	t.Run("PromptResponse/EndTurn", func(t *testing.T) {
		assertSchemaValid(t, "PromptResponse", types.PromptResponse{
			StopReason: types.StopReasonEndTurn,
		})
	})

	t.Run("PromptResponse/Cancelled", func(t *testing.T) {
		assertSchemaValid(t, "PromptResponse", types.PromptResponse{
			StopReason: types.StopReasonCancelled,
		})
	})

	t.Run("SetSessionModeResponse", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionModeResponse", types.SetSessionModeResponse{})
	})

	t.Run("SetSessionConfigOptionResponse", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionConfigOptionResponse", types.SetSessionConfigOptionResponse{
			ConfigOptions: []types.SessionConfigOption{},
		})
	})

	t.Run("AuthenticateResponse", func(t *testing.T) {
		assertSchemaValid(t, "AuthenticateResponse", types.AuthenticateResponse{})
	})
}

func TestSchemaSessionUpdates(t *testing.T) {
	t.Run("AgentMessageChunk", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.AgentMessageChunk("hello world"))
	})

	t.Run("AgentThoughtChunk", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.AgentThoughtChunk("thinking..."))
	})

	t.Run("ToolCall", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.ToolCallStart(
			"call_1",
			"Reading main.go",
			types.ToolKindRead,
			[]types.ToolCallLocation{{Path: "/project/main.go"}},
			map[string]any{"path": "/project/main.go"},
		))
	})

	t.Run("ToolCallUpdate/Completed", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.ToolCallUpdate(
			"call_1",
			types.ToolCallStatusCompleted,
			[]types.ToolCallContent{types.TextToolContent("file contents")},
			map[string]any{"content": "file contents"},
		))
	})

	t.Run("ToolCallUpdate/Failed", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.ToolCallUpdate(
			"call_1",
			types.ToolCallStatusFailed,
			[]types.ToolCallContent{types.TextToolContent("permission denied")},
			map[string]any{"error": "permission denied"},
		))
	})

	t.Run("ToolCallUpdate/InProgressNoContent", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.ToolCallUpdate(
			"call_1",
			types.ToolCallStatusInProgress,
			nil,
			nil,
		))
	})

	t.Run("AvailableCommandsUpdate", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", types.AvailableCommandsUpdate([]types.Command{
			{Name: "commit", Description: "Create a git commit"},
			{Name: "review", Description: "Review code changes"},
		}))
	})
}

func TestSchemaSessionNotification(t *testing.T) {
	assertSchemaValid(t, "SessionNotification", types.SessionNotification{
		SessionID: "sess_123",
		Update:    types.AgentMessageChunk("hello"),
	})
}

func TestSchemaPermission(t *testing.T) {
	t.Run("RequestPermissionRequest", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionRequest", types.RequestPermissionRequest{
			SessionID: "sess_1",
			ToolCall: types.ToolCallDetail{
				ToolCallID: "call_1",
				Title:      ptrTo("write_file"),
				Kind:       ptrTo(types.ToolKindEdit),
				Status:     ptrTo(types.ToolCallStatusPending),
				RawInput:   map[string]any{"path": "/tmp/config.json"},
			},
			Options: []types.PermissionOption{
				{OptionID: "allow", Name: "Allow", Kind: types.PermissionOptionKindAllowOnce},
				{OptionID: "reject", Name: "Reject", Kind: types.PermissionOptionKindRejectOnce},
			},
		})
	})

	t.Run("RequestPermissionRequest/WithPreview", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionRequest", types.RequestPermissionRequest{
			SessionID: "sess_1",
			ToolCall: types.ToolCallDetail{
				ToolCallID: "call_1",
				Title:      ptrTo("edit_file"),
				Kind:       ptrTo(types.ToolKindEdit),
				Status:     ptrTo(types.ToolCallStatusPending),
				RawInput:   map[string]any{"path": "test.txt", "edits": []any{}},
				Preview: map[string]any{
					"path": "test.txt",
					"summary": map[string]any{
						"total_edits":   1,
						"lines_added":   1,
						"lines_removed": 1,
						"net_change":    0,
					},
					"blocks": []any{},
				},
			},
			Options: []types.PermissionOption{
				{OptionID: "allow", Name: "Allow", Kind: types.PermissionOptionKindAllowOnce},
				{OptionID: "reject", Name: "Reject", Kind: types.PermissionOptionKindRejectOnce},
			},
		})
	})

	t.Run("RequestPermissionResponse/Selected", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionResponse", types.RequestPermissionResponse{
			Outcome: types.PermissionSelected("allow"),
		})
	})

	t.Run("RequestPermissionResponse/Cancelled", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionResponse", types.RequestPermissionResponse{
			Outcome: types.PermissionCancelled(),
		})
	})
}

func TestSchemaRequests(t *testing.T) {
	t.Run("InitializeRequest", func(t *testing.T) {
		assertSchemaValid(t, "InitializeRequest", types.InitializeRequest{
			ProtocolVersion: 1,
		})
	})

	t.Run("NewSessionRequest", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionRequest", types.NewSessionRequest{
			Cwd:        "/home/user/project",
			McpServers: []types.McpServer{},
		})
	})

	t.Run("NewSessionRequest/WithMeta", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionRequest", types.NewSessionRequest{
			Cwd:        "/tmp",
			McpServers: []types.McpServer{},
			Meta:       map[string]any{"agentName": "myagent"},
		})
	})

	t.Run("PromptRequest", func(t *testing.T) {
		assertSchemaValid(t, "PromptRequest", types.PromptRequest{
			SessionID: "sess_1",
			Prompt:    []types.ContentBlock{types.TextBlock("Fix the bug")},
		})
	})

	t.Run("CancelNotification", func(t *testing.T) {
		assertSchemaValid(t, "CancelNotification", types.CancelNotification{
			SessionID: "sess_1",
		})
	})

	t.Run("LoadSessionRequest", func(t *testing.T) {
		assertSchemaValid(t, "LoadSessionRequest", types.LoadSessionRequest{
			SessionID:  "sess_1",
			Cwd:        "/project",
			McpServers: []types.McpServer{},
		})
	})

	t.Run("SetSessionModeRequest", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionModeRequest", types.SetSessionModeRequest{
			SessionID: "sess_1",
			ModeID:    "plan",
		})
	})

	t.Run("SetSessionConfigOptionRequest", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionConfigOptionRequest", types.SetSessionConfigOptionRequest{
			SessionID: "sess_1",
			ConfigID:  "model",
			Value:     "gpt-4",
		})
	})
}

func TestSchemaToolCallContent(t *testing.T) {
	assertSchemaValid(t, "ToolCallContent", types.TextToolContent("hello world"))
}

func ptrTo[T any](v T) *T { return &v }
