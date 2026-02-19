package protocol

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/erg0nix/kontekst/internal/core"
)

// ToolExecutor delegates tool execution to the ACP client based on its declared capabilities.
type ToolExecutor struct {
	conn      *Connection
	sessionID SessionID
	caps      ClientCapabilities
}

// NewToolExecutor creates an ToolExecutor that routes tool calls over the given connection.
func NewToolExecutor(conn *Connection, sessionID SessionID, caps ClientCapabilities) *ToolExecutor {
	return &ToolExecutor{
		conn:      conn,
		sessionID: sessionID,
		caps:      caps,
	}
}

// ToolDefinitions returns tool definitions based on the client's declared capabilities.
func (e *ToolExecutor) ToolDefinitions() []core.ToolDef {
	var defs []core.ToolDef

	if e.caps.Fs != nil && e.caps.Fs.ReadTextFile {
		defs = append(defs, core.ToolDef{
			Name:        "read_file",
			Description: "Read the contents of a text file. Returns the file content as a string.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to read",
					},
					"line": map[string]any{
						"type":        "integer",
						"description": "Line number to start reading from (1-based)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of lines to read",
					},
				},
				"required": []string{"path"},
			},
		})
	}

	if e.caps.Fs != nil && e.caps.Fs.WriteTextFile {
		defs = append(defs, core.ToolDef{
			Name:        "write_file",
			Description: "Write text content to a file. Creates or overwrites the file at the given path.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to write",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "The text content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		})
	}

	if e.caps.Terminal {
		defs = append(defs, core.ToolDef{
			Name:        "run_command",
			Description: "Execute a shell command and return its output.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The command to execute",
					},
					"args": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Command arguments",
					},
					"cwd": map[string]any{
						"type":        "string",
						"description": "Working directory (absolute path)",
					},
				},
				"required": []string{"command"},
			},
		})
	}

	return defs
}

// Execute runs the named tool by delegating to the appropriate ACP method on the client.
func (e *ToolExecutor) Execute(name string, args map[string]any, ctx context.Context) (string, error) {
	switch name {
	case "read_file":
		return e.executeReadFile(args, ctx)
	case "write_file":
		return e.executeWriteFile(args, ctx)
	case "run_command":
		return e.executeRunCommand(args, ctx)
	default:
		return "", fmt.Errorf("acp executor: unknown tool %q", name)
	}
}

// Preview returns an empty string because ACP-delegated tools do not support previews.
func (e *ToolExecutor) Preview(_ string, _ map[string]any, _ context.Context) (string, error) {
	return "", nil
}

func (e *ToolExecutor) executeReadFile(args map[string]any, ctx context.Context) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return "", fmt.Errorf("acp executor: read_file requires path")
	}

	req := ReadTextFileRequest{
		SessionID: e.sessionID,
		Path:      path,
	}

	if line, ok := toUint32(args["line"]); ok {
		req.Line = &line
	}
	if limit, ok := toUint32(args["limit"]); ok {
		req.Limit = &limit
	}

	result, err := e.conn.Request(ctx, MethodFsReadTextFile, req)
	if err != nil {
		return "", fmt.Errorf("acp executor: fs/read_text_file: %w", err)
	}

	var resp ReadTextFileResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("acp executor: unmarshal read response: %w", err)
	}

	return resp.Content, nil
}

func (e *ToolExecutor) executeWriteFile(args map[string]any, ctx context.Context) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return "", fmt.Errorf("acp executor: write_file requires path")
	}

	content, _ := args["content"].(string)

	req := WriteTextFileRequest{
		SessionID: e.sessionID,
		Path:      path,
		Content:   content,
	}

	_, err := e.conn.Request(ctx, MethodFsWriteTextFile, req)
	if err != nil {
		return "", fmt.Errorf("acp executor: fs/write_text_file: %w", err)
	}

	return fmt.Sprintf("wrote to %s", path), nil
}

func (e *ToolExecutor) executeRunCommand(args map[string]any, ctx context.Context) (string, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return "", fmt.Errorf("acp executor: run_command requires command")
	}

	req := CreateTerminalRequest{
		SessionID: e.sessionID,
		Command:   command,
	}

	if rawArgs, ok := args["args"]; ok {
		if strArgs, ok := toStringSlice(rawArgs); ok {
			req.Args = strArgs
		}
	}
	if cwd, ok := args["cwd"].(string); ok {
		req.Cwd = cwd
	}

	result, err := e.conn.Request(ctx, MethodTerminalCreate, req)
	if err != nil {
		return "", fmt.Errorf("acp executor: terminal/create: %w", err)
	}

	var createResp CreateTerminalResponse
	if err := json.Unmarshal(result, &createResp); err != nil {
		return "", fmt.Errorf("acp executor: unmarshal create response: %w", err)
	}

	termReq := WaitForExitRequest{
		SessionID:  e.sessionID,
		TerminalID: createResp.TerminalID,
	}

	exitResult, err := e.conn.Request(ctx, MethodTerminalWait, termReq)
	if err != nil {
		return "", fmt.Errorf("acp executor: terminal/wait_for_exit: %w", err)
	}

	var exitResp WaitForExitResponse
	if err := json.Unmarshal(exitResult, &exitResp); err != nil {
		return "", fmt.Errorf("acp executor: unmarshal exit response: %w", err)
	}

	outReq := TerminalOutputRequest{
		SessionID:  e.sessionID,
		TerminalID: createResp.TerminalID,
	}

	outResult, err := e.conn.Request(ctx, MethodTerminalOutput, outReq)
	if err != nil {
		return "", fmt.Errorf("acp executor: terminal/output: %w", err)
	}

	var outResp TerminalOutputResponse
	if err := json.Unmarshal(outResult, &outResp); err != nil {
		return "", fmt.Errorf("acp executor: unmarshal output response: %w", err)
	}

	_, _ = e.conn.Request(ctx, MethodTerminalRelease, ReleaseTerminalRequest{
		SessionID:  e.sessionID,
		TerminalID: createResp.TerminalID,
	})

	return formatCommandOutput(outResp, exitResp), nil
}

func formatCommandOutput(out TerminalOutputResponse, exit WaitForExitResponse) string {
	output := out.Output

	if exit.ExitCode != nil && *exit.ExitCode != 0 {
		output += fmt.Sprintf("\n[exit code: %d]", *exit.ExitCode)
	}
	if exit.Signal != nil {
		output += fmt.Sprintf("\n[signal: %s]", *exit.Signal)
	}
	if out.Truncated {
		output += "\n[output truncated]"
	}

	return output
}

func toUint32(v any) (uint32, bool) {
	switch n := v.(type) {
	case float64:
		if n >= 0 {
			return uint32(n), true
		}
	case json.Number:
		if i, err := n.Int64(); err == nil && i >= 0 {
			return uint32(i), true
		}
	}
	return 0, false
}

func toStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok {
		return nil, false
	}

	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}
