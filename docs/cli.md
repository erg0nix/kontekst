# CLI Reference

## Root Command

```
kontekst [prompt]
```

Sends a prompt to the agent and streams the response. If no daemon is running, prints an error with instructions to start one.

```bash
kontekst "explain this error"
kontekst "refactor the auth module to use interfaces"
```

If the prompt starts with `/`, it is treated as a skill invocation (see [Skills](#skill-invocation)).

## Commands

### `start`

Start the kontekst daemon.

```bash
kontekst start
kontekst start --foreground
kontekst start --daemon-bin /usr/local/bin/kontekst-daemon
```

| Flag | Description |
|------|-------------|
| `--foreground` | Run the daemon in the foreground with stdout/stderr attached. Without this flag, the daemon runs in the background and logs to `~/.kontekst/daemon.log`. |
| `--daemon-bin` | Path to the daemon binary. Defaults to `kontekst-daemon` in the same directory as the CLI binary, falling back to `$PATH`. |

If a daemon is already running at the configured address, the command prints a message and exits.

### `stop`

Stop a running daemon.

```bash
kontekst stop
```

Sends a `Shutdown` RPC to the daemon. Times out after 5 seconds.

### `ps`

Show daemon status.

```bash
kontekst ps
```

Output includes:
- Daemon address, bind address, uptime, start time
- Data directory

### `agents`

List available agents.

```bash
kontekst agents
```

Scans `~/.kontekst/agents/` for agent directories. Each agent directory can contain:
- `config.toml` - Agent configuration (model, sampling parameters)
- `agent.md` - System prompt

Output is a table with columns: NAME, DISPLAY NAME, PROMPT, CONFIG.

### `llama start`

Start llama-server with hardcoded defaults (127.0.0.1:8080, ~/models, 99 GPU layers).

```bash
kontekst llama start
kontekst llama start --background
kontekst llama start --bin /usr/local/bin/llama-server
```

| Flag | Description |
|------|-------------|
| `--background` | Run llama-server in the background (detached). |
| `--bin` | Path to the llama-server binary. Defaults to `llama-server` on `$PATH`. |

### `llama stop`

Stop a running llama-server.

```bash
kontekst llama stop
```

### `session set-agent`

Set the default agent for the current session.

```bash
kontekst session set-agent myagent
```

Requires an active session (run a prompt first to create one). The default agent is stored in a `.meta.json` file alongside the session's JSONL file and is used for subsequent runs in that session unless overridden with `--agent`.

## Global Flags

These flags are available on all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file. Defaults to `~/.kontekst/config.toml`. |
| `--server` | | gRPC server address. Overrides the `bind` value from config. |
| `--auto-approve` | | Auto-approve all tool calls without prompting. |
| `--session` | | Session ID to reuse. Overrides the stored active session. |
| `--agent` | | Agent to use for this run. Overrides the session's default agent. |

## Skill Invocation

Prompts starting with `/` invoke a skill:

```bash
kontekst "/summarize path/to/file.go"
kontekst "/review"
```

The first word after `/` is the skill name. Everything after that is passed as arguments. Skills are loaded from `~/.kontekst/skills/` as markdown files with optional TOML frontmatter.

Skill arguments are available in the skill template as `$ARGUMENTS` (the full string) or `$0`, `$1`, etc. (positional, space-separated, with quote support).

## Tool Approval Workflow

When the agent proposes a tool call, the CLI displays:
1. The tool name and its arguments (JSON)
2. A preview (if the tool implements the `Previewer` interface)
3. A prompt: `approve? [y/N]`

Typing `y` or `Y` approves the tool. Anything else (including pressing Enter) denies it. Denied tools end the run.

With `--auto-approve`, all tools are approved automatically without prompting.

## Sessions

Sessions are created automatically on the first run and reused for subsequent runs. The active session ID is stored in `~/.kontekst/active_session`.

To use a specific session:

```bash
kontekst --session abc123 "continue where we left off"
```

Session history (messages from previous runs) is loaded at the start of each run, subject to the context window's token budget. Recent messages are prioritized.
