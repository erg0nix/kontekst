# gRPC API Reference

Defined in `proto/kontekst.proto`. Package: `kontekst`.

## AgentService

### `Run` (bidirectional streaming)

```protobuf
rpc Run(stream RunCommand) returns (stream RunEvent);
```

The client sends commands, the server sends events. A run starts when the client sends `StartRunCommand` and ends when the server sends `RunCompletedEvent`, `RunFailedEvent`, or `RunCancelledEvent`.

### Client Commands (`RunCommand`)

A `RunCommand` is a `oneof` containing exactly one of:

#### `StartRunCommand`

Starts a new agent run.

| Field | Type | Description |
|-------|------|-------------|
| `prompt` | string | The user prompt. Empty if using a skill invocation. |
| `session_id` | string | Session to continue. Empty creates a new session. |
| `agent_name` | string | Agent to use (e.g., `"default"`). |
| `working_dir` | string | Working directory for file tools. |
| `skill` | SkillInvocation | Skill to invoke instead of a prompt. |

#### `ApproveToolCommand`

Approves a proposed tool call.

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | ID of the tool call to approve. |

#### `DenyToolCommand`

Denies a proposed tool call.

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | ID of the tool call to deny. |
| `reason` | string | Reason for denial. |

#### `CancelRunCommand`

Cancels the current run. No fields.

### Server Events (`RunEvent`)

A `RunEvent` is a `oneof` containing exactly one of:

#### `RunStartedEvent`

Sent once after the run begins.

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | Unique run identifier. |
| `session_id` | string | Session ID (may be newly created). |
| `agent_name` | string | Agent handling the run. |

#### `TokenDeltaEvent`

Streaming token from the LLM response.

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Token text fragment. |

#### `ReasoningDeltaEvent`

Streaming reasoning token (for models that produce chain-of-thought).

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Reasoning text fragment. |

#### `TurnCompletedEvent`

Sent after each LLM turn completes.

| Field | Type | Description |
|-------|------|-------------|
| `content` | string | Full response content. |
| `reasoning` | string | Full reasoning content. |
| `context` | ContextSnapshot | Token usage snapshot. |

#### `ToolsProposedEvent`

The LLM wants to call one or more tools. The client must respond with `ApproveToolCommand` or `DenyToolCommand` for each call.

| Field | Type | Description |
|-------|------|-------------|
| `calls` | repeated ProposedToolCall | List of proposed tool calls. |

#### `ToolExecutionStartedEvent`

A tool has started executing.

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | ID of the tool call. |

#### `ToolExecutionCompletedEvent`

A tool finished successfully.

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | ID of the tool call. |
| `output` | string | Tool output text. |

#### `ToolExecutionFailedEvent`

A tool failed.

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | ID of the tool call. |
| `error` | string | Error message. |

#### `ToolsCompletedEvent`

All proposed tools in this turn have been processed. No fields.

#### `RunCompletedEvent`

The run finished normally.

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | Run identifier. |
| `content` | string | Final response content. |
| `reasoning` | string | Final reasoning content. |

#### `RunCancelledEvent`

The run was cancelled (by client or due to a denied tool).

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | Run identifier. |

#### `RunFailedEvent`

The run failed due to an error.

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | string | Run identifier. |
| `error` | string | Error message. |

#### `ContextSnapshotEvent`

Token usage snapshot, sent after each turn.

| Field | Type | Description |
|-------|------|-------------|
| `context` | ContextSnapshot | Token usage data. |

## DaemonService

### `GetStatus`

```protobuf
rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
```

Returns daemon status. `GetStatusRequest` has no fields.

**`GetStatusResponse`**:

| Field | Type | Description |
|-------|------|-------------|
| `bind` | string | gRPC listen address. |
| `endpoint` | string | llama-server HTTP endpoint. |
| `model_dir` | string | Model directory path. |
| `llama_server_healthy` | bool | Whether llama-server health check passes. |
| `llama_server_running` | bool | Whether llama-server process is running. |
| `llama_server_pid` | int32 | llama-server process ID (0 if not running). |
| `uptime_seconds` | int64 | Daemon uptime in seconds. |
| `started_at_rfc3339` | string | Daemon start time in RFC 3339 format. |
| `data_dir` | string | Data directory path. |

### `Shutdown`

```protobuf
rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
```

Gracefully shuts down the daemon. `ShutdownRequest` has no fields.

**`ShutdownResponse`**:

| Field | Type | Description |
|-------|------|-------------|
| `message` | string | Confirmation message. |

## Supporting Messages

### `SkillInvocation`

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Skill name (e.g., `"summarize"`). |
| `arguments` | string | Raw argument string. |

### `ProposedToolCall`

| Field | Type | Description |
|-------|------|-------------|
| `call_id` | string | Unique call identifier. |
| `name` | string | Tool name (e.g., `"read_file"`). |
| `arguments_json` | string | Tool arguments as JSON string. |
| `preview` | string | Preview output (empty if tool has no previewer). |

### `ContextSnapshot`

| Field | Type | Description |
|-------|------|-------------|
| `context_size` | int32 | Total context window size in tokens. |
| `system_tokens` | int32 | Tokens used by the system prompt. |
| `tool_tokens` | int32 | Tokens used by tool definitions. |
| `history_tokens` | int32 | Tokens used by session history. |
| `memory_tokens` | int32 | Tokens used by current run messages. |
| `total_tokens` | int32 | Total tokens in use. |
| `remaining_tokens` | int32 | Tokens remaining in the context window. |
| `history_messages` | int32 | Number of messages from session history. |
| `memory_messages` | int32 | Number of messages from the current run. |
| `total_messages` | int32 | Total message count (including system). |
| `history_budget` | int32 | Token budget allocated to history. |
| `messages` | repeated MessageStats | Per-message token breakdown. |

### `MessageStats`

| Field | Type | Description |
|-------|------|-------------|
| `role` | string | Message role: `"system"`, `"user"`, `"assistant"`, or `"tool"`. |
| `tokens` | int32 | Token count for this message. |
| `source` | string | `"system"`, `"history"`, or `"memory"`. |
