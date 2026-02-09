# Architecture & Internals

## Overview

Kontekst is a local-first AI agent with a daemon/client split. The CLI communicates with the daemon over gRPC. The daemon manages agent execution, tool calls, and LLM communication via a local llama-server instance.

```
┌───────┐       gRPC        ┌──────────┐       HTTP        ┌──────────────┐
│  CLI  │◄────────────────►│  Daemon   │◄────────────────►│ llama-server │
└───────┘                   └──────────┘                   └──────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
               ┌────▼───┐  ┌────▼───┐  ┌────▼────┐
               │ Tools  │  │Sessions│  │ Skills  │
               └────────┘  └────────┘  └─────────┘
```

The CLI sends a prompt (or skill invocation) to the daemon, which runs the agent loop. When the agent proposes tool calls, the daemon streams them back to the CLI for user approval. Approved tools execute on the daemon side, and results feed back into the LLM.

## Agent Loop

The core execution cycle in `internal/agent/agent.go`:

1. **Start run** - Generate a run ID, compute token budgets, load session history
2. **Build context** - Assemble system prompt + history + current run messages
3. **Call LLM** - Send context and tool definitions to llama-server via `GenerateChat`
4. **Check for tool calls** - If the response contains no tool calls, emit the final response and end
5. **Propose tools** - Send proposed tool calls (with previews) to the client for approval
6. **Collect approvals** - Wait for approve/deny decisions for each tool call
7. **Execute tools** - Run approved tools, record results as tool-role messages
8. **Loop** - Go back to step 2 with tool results added to context

If any tool is denied, the run ends after executing the approved ones. If the LLM returns a response with no tool calls, the run completes.

## Package Map

### Layer 0: `internal/core`

Shared types with zero internal dependencies. Everything else imports this package.

- `Message`, `ToolCall`, `ToolResult`, `ToolDef` - conversation primitives
- `ChatResponse`, `Usage` - LLM response types
- `ContextSnapshot`, `MessageStats` - context window observability
- `SamplingConfig` - temperature, top_p, top_k, repeat_penalty, max_tokens
- `SkillMetadata` - skill name and path
- `SessionID`, `RunID` - typed identifiers

### Layer 1: `internal/config`

TOML configuration loading. `Config` struct covers daemon bind address, LLM endpoint, model directory, context size, GPU layers, tool limits.

Default config path: `~/.kontekst/config.toml`. Created automatically on first use.

### Layer 1: `internal/config/agents`

Per-agent configuration. Each agent lives in `~/.kontekst/agents/<name>/` with:

- `config.toml` - model filename, sampling parameters, display name, tool_role flag
- `agent.md` - system prompt

A `default` agent is auto-created if none exists.

### Layer 2: `internal/providers`

LLM backend abstraction. The `Provider` interface:

- `GenerateChat(messages, tools, sampling, model, useToolRole)` - send a chat completion request
- `CountTokens(text)` - estimate token count
- `ConcurrencyLimit()` - max parallel requests

Currently one implementation: llama-server (HTTP API). `SingleProviderRouter` wraps a `Provider` with a concurrency semaphore.

### Layer 2: `internal/context`

Conversation context management. `ContextWindow` interface manages the token budget across:

- **System prompt** - agent system prompt + active skill metadata
- **Tool definitions** - JSON-serialized tool schemas
- **History** - messages from previous runs (loaded from JSONL session file)
- **Memory** - messages from the current run

Token budgeting: `history_budget = context_size - system_tokens - tool_tokens - user_prompt_tokens`. History is loaded from the tail of the session file, fitting as many recent messages as the budget allows.

### Layer 2: `internal/sessions`

Session persistence. `FileSessionService` manages:

- Session creation (generates UUID, creates JSONL file)
- Session ensure (creates file if missing)
- Default agent per session (stored in `<session_id>.meta.json`)

Session data lives in `~/.kontekst/sessions/`.

### Layer 2: `internal/tools`

Tool registry and interfaces. The `Tool` interface:

- `Name()`, `Description()`, `Parameters()` - metadata exposed to the LLM
- `RequiresApproval()` - whether the user must approve execution
- `Execute(args, ctx)` - perform the action

Optional `Previewer` interface adds `Preview(args, ctx)` for showing what a tool will do before approval.

`Registry` provides thread-safe tool registration, execution dispatch, and definition export.

### Layer 2: `internal/skills`

Reusable prompt templates. Skills are markdown files with optional TOML frontmatter (`+++` delimiters):

```markdown
+++
name = "summarize"
description = "Summarize a file"
+++

Summarize the following file: $ARGUMENTS
```

Frontmatter fields:
- `name` - skill name (defaults to filename or directory name)
- `description` - shown in skill listings
- `disable_model_invocation` - hide from LLM's available skills
- `user_invocable` - whether users can invoke with `/name` (default: true)

Argument substitution: `$ARGUMENTS` for the full string, `$0`, `$1`, etc. for positional args (quote-aware splitting).

### Layer 3: `internal/tools/builtin`

Built-in tool implementations:

- `read_file` - read file contents
- `write_file` - write/create files
- `edit_file` - search-and-replace edits with preview
- `list_files` - list directory contents
- `web_fetch` - fetch URL content
- `skill` - invoke a skill by name (does not require approval)

The first five are registered via `RegisterAll()`. The `skill` tool is registered separately with a reference to the skill registry. File tools resolve paths relative to the working directory and reject path traversal (`..`).

### Layer 4: `internal/agent`

Agent orchestration. `Agent` struct wires together a provider, tool executor, and context window. `Agent.Run()` launches the agent loop in a goroutine, returning command/event channels.

Also contains the agent `Registry` for discovering and loading agent configurations from `~/.kontekst/agents/`.

### Layer 5: `internal/grpc` + `internal/grpc/pb`

gRPC transport layer. `pb` contains generated protobuf code. The service implementations translate between gRPC streams and agent command/event channels.

Two services:
- `AgentService.Run` - bidirectional streaming for agent execution
- `DaemonService` - `GetStatus` and `Shutdown` unary RPCs

### Layer 6: `cmd/daemon` + `cmd/cli`

Executables. The daemon starts the gRPC server and manages llama-server. The CLI parses commands, connects to the daemon, and handles the interactive tool approval workflow.

## Context Management

The context window uses a dual-layer memory model:

**History** (persistent): Messages from previous runs, stored in a JSONL session file. On each run start, the tail of the file is loaded up to the computed token budget. Older messages are dropped first.

**Memory** (ephemeral): Messages from the current run, kept in memory. Every message (user prompts, assistant responses, tool calls, tool results) is both appended to the session file and held in memory.

Context assembly for each LLM call:

```
[system message] + [history messages] + [memory messages]
```

The history budget shrinks as memory grows, but history is only loaded once at run start. This means long runs accumulate memory messages without re-trimming history mid-run.

## Tool System

### Registration

Tools register in `internal/tools/builtin/builtin.go` via `RegisterAll()`. Each tool is a struct implementing the `Tool` interface.

### Execution Flow

1. LLM response includes tool calls (name + JSON arguments)
2. Agent builds `ProposedToolCall` list with previews
3. Proposals are sent to the client via `ToolsProposedEvent`
4. Client responds with approve/deny for each call
5. Approved tools execute via `Registry.Execute()`
6. Results are added as tool-role messages to context
7. Agent loops back to the LLM with updated context

### Preview

Tools implementing `Previewer` can show what they'll do before execution. For example, `edit_file` shows a diff preview. Previews are computed before sending proposals to the client.

## Session Persistence

Sessions use JSONL (JSON Lines) files. Each line is a JSON-encoded `Message`:

```json
{"role":"user","content":"explain this","tokens":15}
{"role":"assistant","content":"This is...","tokens":42}
```

Files live at `~/.kontekst/sessions/<session_id>.jsonl` with companion `<session_id>.meta.json` files for metadata (currently just the default agent name).

History loading reads the file backwards in 8KB chunks, parsing messages from newest to oldest until the token budget is exhausted.

## Configuration

### Global Config (`~/.kontekst/config.toml`)

```toml
bind = ":50051"
endpoint = "http://127.0.0.1:8080"
model_dir = "~/models"
context_size = 4096
gpu_layers = 0

[tools]
working_dir = ""
[tools.file]
max_size_bytes = 10485760
[tools.web]
timeout_seconds = 30
max_response_bytes = 5242880
```

| Setting | Default | Description |
|---------|---------|-------------|
| `bind` | `:50051` | gRPC listen address |
| `endpoint` | `http://127.0.0.1:8080` | llama-server URL |
| `model_dir` | `~/models` | Directory with GGUF model files |
| `context_size` | 4096 | Token context window |
| `gpu_layers` | 0 | GPU layers for llama-server |

### Per-Agent Config (`~/.kontekst/agents/<name>/config.toml`)

```toml
name = "Default Assistant"
model = "gpt-oss-20b-Q4_K_M.gguf"
tool_role = false

[sampling]
temperature = 0.7
top_p = 0.9
top_k = 40
repeat_penalty = 1.1
max_tokens = 4096
```

| Setting | Description |
|---------|-------------|
| `name` | Display name for the agent |
| `model` | GGUF model filename (within `model_dir`) |
| `tool_role` | Use `tool` role for tool results instead of embedding in `user` messages |
| `sampling.*` | LLM sampling parameters |

### Data Directory Layout

```
~/.kontekst/
├── config.toml
├── active_session
├── daemon.log
├── agents/
│   └── default/
│       ├── config.toml
│       └── agent.md
├── skills/
│   ├── summarize.md
│   └── review/
│       └── SKILL.md
├── sessions/
│   ├── <session_id>.jsonl
│   └── <session_id>.meta.json
└── daemon.log
```
