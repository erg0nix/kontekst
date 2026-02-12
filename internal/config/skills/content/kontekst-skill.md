+++
name = "kontekst"
description = "Kontekst self-documentation: architecture, usage, configuration"
disable_model_invocation = true
+++

# Kontekst

Kontekst is a local-first AI agent framework with a single-binary client/server architecture.

## Quick Start

```bash
# Start the LLM backend
kontekst llama start --background

# Start the server
kontekst start

# Run a prompt
kontekst "your prompt here"

# Run with auto-approval (skip tool confirmation)
kontekst --auto-approve "your prompt"

# Check status
kontekst ps

# Stop everything
kontekst stop
kontekst llama stop
```

## Architecture

```
CLI --[ACP/TCP]--> Server --[HTTP]--> llama-server (local LLM)
         ^                   |
         |                   v
    Tool Approval       Tool Execution
```

- **Server** (`kontekst start`): TCP server hosting the agent runtime. Manages tool execution and session persistence.
- **CLI** (`kontekst [prompt]`): ACP client that streams agent events and handles tool approval prompts.
- **Stdio mode** (`kontekst acp`): ACP over stdin/stdout for editor integration.

The protocol is ACP (Agent Client Protocol) over JSON-RPC 2.0 with line-delimited JSON messages.

## Agents

Agents live in `~/.kontekst/agents/<name>/`. Each agent has:
- `config.toml` — provider, sampling, and context settings
- `agent.md` — system prompt

Bundled agents:
- **default** — balanced general-purpose assistant (temperature 0.7)
- **coder** — precise coding assistant (temperature 0.3)
- **fantasy** — creative fantasy writing partner (temperature 0.9)

Select an agent: `kontekst --agent coder "fix the bug"`

### Agent Config Example

```toml
name = "My Agent"
context_size = 4096
tool_role = false

[provider]
endpoint = "http://127.0.0.1:8080"
model = "my-model.gguf"

[sampling]
temperature = 0.7
top_p = 0.9
top_k = 40
repeat_penalty = 1.1
max_tokens = 4096
```

## Skills

Skills are reusable prompt templates. They live in `~/.kontekst/skills/`.

A skill is either:
- A directory with a `SKILL.md` file
- A standalone `.md` file

Skills support TOML frontmatter (`+++` delimited) for metadata:
```
+++
name = "my-skill"
description = "What this skill does"
disable_model_invocation = true
+++

Skill content here. Use $ARGUMENTS for user input.
```

The model can invoke skills via the `skill` tool. Users invoke skills with `/skill-name`.

## Commands

Commands are executable scripts. They live in `~/.kontekst/commands/<name>/`.

Each command has:
- `command.toml` — metadata, arguments, runtime
- `run.sh` or `run.py` — the script to execute

```toml
name = "my-command"
description = "What this command does"
runtime = "bash"
working_dir = "agent"
timeout = 30

[[arguments]]
name = "pattern"
type = "string"
description = "Search pattern"
required = true
```

The model invokes commands via the `command` tool.

## Built-in Tools

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write content to a file |
| `edit_file` | Apply targeted edits to a file |
| `list_files` | List files in a directory |
| `web_fetch` | Fetch content from a URL |
| `run_command` | Execute a shell command |
| `skill` | Invoke a skill |
| `command` | Invoke a command |

All tools require user approval before execution.

## Sessions

Sessions persist conversation history across runs. State lives in `~/.kontekst/sessions/`.

- Each session stores messages in append-only files
- The active session ID is tracked in `~/.kontekst/active_session`
- Sessions can be loaded or created via ACP protocol messages

## Data Layout

```
~/.kontekst/
  config.toml          # Server configuration
  server.pid           # Server PID (when backgrounded)
  server.log           # Server output (when backgrounded)
  active_session       # Current session ID
  agents/              # Agent configurations
    default/
    coder/
    fantasy/
  skills/              # Skill definitions
  commands/            # Command definitions
  sessions/            # Session persistence
```
