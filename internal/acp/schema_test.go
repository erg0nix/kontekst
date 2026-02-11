package acp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

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
		assertSchemaValid(t, "InitializeResponse", InitializeResponse{
			ProtocolVersion: 1,
			AgentCapabilities: AgentCapabilities{
				LoadSession: true,
			},
			AgentInfo: &Implementation{
				Name:    "kontekst",
				Title:   "Kontekst",
				Version: "0.1.0",
			},
			AuthMethods: []AuthMethod{},
		})
	})

	t.Run("NewSessionResponse", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionResponse", NewSessionResponse{
			SessionID: "sess_123",
		})
	})

	t.Run("LoadSessionResponse", func(t *testing.T) {
		assertSchemaValid(t, "LoadSessionResponse", LoadSessionResponse{
			SessionID: "sess_123",
		})
	})

	t.Run("PromptResponse/EndTurn", func(t *testing.T) {
		assertSchemaValid(t, "PromptResponse", PromptResponse{
			StopReason: StopReasonEndTurn,
		})
	})

	t.Run("PromptResponse/Cancelled", func(t *testing.T) {
		assertSchemaValid(t, "PromptResponse", PromptResponse{
			StopReason: StopReasonCancelled,
		})
	})

	t.Run("SetSessionModeResponse", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionModeResponse", SetSessionModeResponse{})
	})

	t.Run("SetSessionConfigOptionResponse", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionConfigOptionResponse", SetSessionConfigOptionResponse{
			ConfigOptions: []SessionConfigOption{},
		})
	})

	t.Run("AuthenticateResponse", func(t *testing.T) {
		assertSchemaValid(t, "AuthenticateResponse", AuthenticateResponse{})
	})
}

func TestSchemaSessionUpdates(t *testing.T) {
	t.Run("AgentMessageChunk", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", AgentMessageChunk("hello world"))
	})

	t.Run("AgentThoughtChunk", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", AgentThoughtChunk("thinking..."))
	})

	t.Run("ToolCall", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", ToolCallStart(
			"call_1",
			"Reading main.go",
			ToolKindRead,
			[]ToolCallLocation{{Path: "/project/main.go"}},
			map[string]any{"path": "/project/main.go"},
		))
	})

	t.Run("ToolCallUpdate/Completed", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", ToolCallUpdate(
			"call_1",
			ToolCallStatusCompleted,
			[]ToolCallContent{TextToolContent("file contents")},
			map[string]any{"content": "file contents"},
		))
	})

	t.Run("ToolCallUpdate/Failed", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", ToolCallUpdate(
			"call_1",
			ToolCallStatusFailed,
			[]ToolCallContent{TextToolContent("permission denied")},
			map[string]any{"error": "permission denied"},
		))
	})

	t.Run("ToolCallUpdate/InProgressNoContent", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", ToolCallUpdate(
			"call_1",
			ToolCallStatusInProgress,
			nil,
			nil,
		))
	})

	t.Run("AvailableCommandsUpdate", func(t *testing.T) {
		assertSchemaValid(t, "SessionUpdate", AvailableCommandsUpdate([]Command{
			{Name: "commit", Description: "Create a git commit"},
			{Name: "review", Description: "Review code changes"},
		}))
	})
}

func TestSchemaSessionNotification(t *testing.T) {
	assertSchemaValid(t, "SessionNotification", SessionNotification{
		SessionID: "sess_123",
		Update:    AgentMessageChunk("hello"),
	})
}

func TestSchemaPermission(t *testing.T) {
	t.Run("RequestPermissionRequest", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionRequest", RequestPermissionRequest{
			SessionID: "sess_1",
			ToolCall: ToolCallDetail{
				ToolCallID: "call_1",
				Title:      ptrTo("write_file"),
				Kind:       ptrTo(ToolKindEdit),
				Status:     ptrTo(ToolCallStatusPending),
				RawInput:   map[string]any{"path": "/tmp/config.json"},
			},
			Options: []PermissionOption{
				{OptionID: "allow", Name: "Allow", Kind: PermissionOptionKindAllowOnce},
				{OptionID: "reject", Name: "Reject", Kind: PermissionOptionKindRejectOnce},
			},
		})
	})

	t.Run("RequestPermissionResponse/Selected", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionResponse", RequestPermissionResponse{
			Outcome: PermissionSelected("allow"),
		})
	})

	t.Run("RequestPermissionResponse/Cancelled", func(t *testing.T) {
		assertSchemaValid(t, "RequestPermissionResponse", RequestPermissionResponse{
			Outcome: PermissionCancelled(),
		})
	})
}

func TestSchemaRequests(t *testing.T) {
	t.Run("InitializeRequest", func(t *testing.T) {
		assertSchemaValid(t, "InitializeRequest", InitializeRequest{
			ProtocolVersion: 1,
		})
	})

	t.Run("NewSessionRequest", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionRequest", NewSessionRequest{
			Cwd:        "/home/user/project",
			McpServers: []McpServer{},
		})
	})

	t.Run("NewSessionRequest/WithMeta", func(t *testing.T) {
		assertSchemaValid(t, "NewSessionRequest", NewSessionRequest{
			Cwd:        "/tmp",
			McpServers: []McpServer{},
			Meta:       map[string]any{"agentName": "myagent"},
		})
	})

	t.Run("PromptRequest", func(t *testing.T) {
		assertSchemaValid(t, "PromptRequest", PromptRequest{
			SessionID: "sess_1",
			Prompt:    []ContentBlock{TextBlock("Fix the bug")},
		})
	})

	t.Run("CancelNotification", func(t *testing.T) {
		assertSchemaValid(t, "CancelNotification", CancelNotification{
			SessionID: "sess_1",
		})
	})

	t.Run("LoadSessionRequest", func(t *testing.T) {
		assertSchemaValid(t, "LoadSessionRequest", LoadSessionRequest{
			SessionID:  "sess_1",
			Cwd:        "/project",
			McpServers: []McpServer{},
		})
	})

	t.Run("SetSessionModeRequest", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionModeRequest", SetSessionModeRequest{
			SessionID: "sess_1",
			ModeID:    "plan",
		})
	})

	t.Run("SetSessionConfigOptionRequest", func(t *testing.T) {
		assertSchemaValid(t, "SetSessionConfigOptionRequest", SetSessionConfigOptionRequest{
			SessionID: "sess_1",
			ConfigID:  "model",
			Value:     "gpt-4",
		})
	})
}

func TestSchemaToolCallContent(t *testing.T) {
	assertSchemaValid(t, "ToolCallContent", TextToolContent("hello world"))
}

func ptrTo[T any](v T) *T { return &v }
