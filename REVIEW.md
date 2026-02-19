# kontekst Restructuring Plan

Review and restructuring plan against `go`, `go-structure`, and `go-cobra` skills.
Branch: `refactor/code-review-fixes`.

---

## The Story Imports Should Tell

Current:
```
cmd/cli → cli → acp → agent → {providers, context, sessions, tools, skills} → core
```

Target:
```
cmd/cli → cli (thin cobra commands)
        → app (server wiring + lifecycle)
        → protocol (JSON-RPC client/server)
        → agent (loop + events)
        → {provider, conversation, session, tool, skill, command}
        → core (conversation primitives: Message, ToolCall, ToolDef, IDs)
```

Each name is a word you understand without context. The import chain reads as a
sentence: the CLI starts the app, which speaks a protocol to the agent, which uses
a provider, manages conversations, persists sessions, executes tools, and loads
skills — all communicating through core conversation primitives.

The `app` package is a new `internal/app/` — it's the server entry point. A
newcomer asking "where does the server run?" finds it immediately.

---

## 1. Package Renames

### 1A. Plural → Singular

**Reference:** package-design.md:13 — "Use singular package names"

| Current | New | Importers |
|---------|-----|-----------|
| `internal/providers` | `internal/provider` | agent |
| `internal/sessions` | `internal/session` | agent, cli |
| `internal/skills` | `internal/skill` | tools/builtin, agent, acp |
| `internal/commands` | `internal/command` | tools/builtin, cli |
| `internal/tools` | `internal/tool` | tools/builtin, agent, cli |
| `internal/tools/builtin` | `internal/tool/builtin` | agent, cli |
| `internal/tools/diff` | `internal/tool/diff` | tool/builtin |
| `internal/tools/hashline` | `internal/tool/hashline` | tool/builtin |
| `internal/config/agents` | `internal/config/agent` | agent, acp, cli |
| `internal/config/commands` | `internal/config/command` | cli |
| `internal/config/skills` | `internal/config/skill` | cli |

Mechanical find-and-replace on import paths and package declarations.
`go build ./...` catches all missed references.

### 1B. Semantic Renames

| Current | New | Why | Importers |
|---------|-----|-----|-----------|
| `internal/context` | `internal/conversation` | Shadows stdlib `context`; forces aliasing in every file that needs both | agent, cli |
| `internal/acp` | `internal/protocol` | Opaque abbreviation; "protocol" is immediately clear | cli |

### 1C. Type Stutter Removal (after renames)

With singular package names, existing type names stutter:

| Current | After Rename | Fix |
|---------|-------------|-----|
| `context.ContextWindow` | `conversation.ContextWindow` | `conversation.Window` |
| `context.ContextService` | `conversation.ContextService` | `conversation.Service` |
| `context.FileContextService` | `conversation.FileContextService` | `conversation.FileService` |
| `sessions.SessionCreator` | `session.SessionCreator` | `session.Creator` |
| `sessions.SessionInfo` | `session.SessionInfo` | `session.Info` |
| `sessions.FileSessionService` | `session.FileSessionService` | `session.FileService` |

**Abbreviation cleanup** (not stutter, but same commit):

| Current | After Rename | Fix | Reason |
|---------|-------------|-----|--------|
| `acp.ACPToolExecutor` | `protocol.ACPToolExecutor` | `protocol.ToolExecutor` | "ACP" is a meaningless abbreviation in the new package |

**Type renames when moving from core** (Phase 2, but listed here for completeness):

| Current | Moves to | New Name | Why rename |
|---------|----------|----------|------------|
| `core.ChatResponse` | `provider` | `provider.Response` | `provider.ChatResponse` stutters |
| `core.Usage` | `provider` | `provider.Usage` | No stutter, keeps name |
| `core.ContextSnapshot` | `conversation` | `conversation.Snapshot` | `conversation.ContextSnapshot` stutters |
| `core.MessageStats` | `conversation` | `conversation.MessageStats` | No stutter, keeps name |

### 1D. `protocol/types/` Subpackage

The 7 `types_*.go` files in `acp` move to `internal/protocol/types/` with
topic-based names:

```
internal/protocol/types/
├── protocol.go      (ProtocolVersion, Method* constants, ErrorCode)
├── capabilities.go  (Initialize req/resp, Implementation, *Capabilities)
├── session.go       (SessionID, NewSession/LoadSession req/resp, McpServer)
├── message.go       (ContentBlock, PromptRequest/Response, StopReason, chunks)
├── tooling.go       (ToolCallID, ToolKind, ToolCallStatus, ToolCallContent)
├── permission.go    (PermissionOption, ToolCallDetail, RequestPermission req/resp)
└── rpc.go           (StatusResponse, ReadTextFile*, WriteTextFile*, Terminal*)
```

All `internal/protocol` files (`client.go`, `server.go`, `executor.go`,
`jsonrpc.go`) update imports from inline types to `protocol/types`.

---

## 2. Core Trimming

**Reference:** package-design.md:242-249 — "Type bucket anti-pattern"

After renames, `core` still has ~20 types. Some belong in their domain packages.

### Import cycle analysis

Current dependency layers (numbers = layer):
```
0: core, config, skill, command, tool/diff, tool/hashline
1: config/agent → core
   provider → core, config
   session → core
   tool → core
2: conversation → core
   tool/builtin → core, config, tool, tool/diff, tool/hashline, skill, command
3: agent → core, config, config/agent, provider, conversation, session, skill, tool, tool/builtin
4: protocol → core, agent, config/agent, skill
5: cli → everything
```

### Types that can move (no new cycles)

| Type | Move to | Why safe |
|------|---------|----------|
| `ChatResponse`, `Usage` | `provider` | Used by `provider` (defines) and `agent` (consumes). `agent` already imports `provider`. No cycle. |
| `ContextSnapshot`, `MessageStats` | `conversation` | Used by `conversation` (builds) and `agent`/`cli` (reads). Both already import `conversation`. No cycle. |
| `IntFromAny` | Inline as unexported helper | Used by `provider/openai.go` and `tool/builtin/builtin.go`. 10 lines, just duplicate. |

### Types that stay in core (would create cycles)

| Type | Why it stays |
|------|-------------|
| `Message`, `Role`, `ToolCall`, `ToolResult` | Used by `provider`, `conversation`, `agent`, `protocol`. These are the shared conversation vocabulary — moving to any one package forces the others to import it. |
| `ToolDef` | Used by `tool`, `provider`, `protocol`, `agent`. Same cross-cutting issue. |
| `SamplingConfig` | Used by `config/agent`, `provider`, `agent`. Moving to `provider` would force `config/agent → provider` (new dep). Moving to `config/agent` would force `provider → config/agent` (new dep). Core is the neutral home. |
| `SkillMetadata` | Used by `conversation`, `tool/builtin`, `agent`. Moving to `skill` would force `conversation → skill` (new dep). Core is cleaner. |
| IDs (`RunID`, `SessionID`, `ToolCallID`, `RequestID`) | Used everywhere. Genuinely cross-cutting identifiers. |

### After trimming, core contains

```
core/
├── message.go  — Message, Role (+ constants), ToolCall, ToolResult
├── tool.go     — ToolDef, SamplingConfig, SkillMetadata
└── id.go       — RunID, SessionID, ToolCallID, RequestID + generators
```

~14 types. All genuinely cross-cutting conversation/protocol primitives that
cannot live elsewhere without creating import cycles. Add a package doc comment:

```go
// Package core defines the shared conversation primitives used across all
// layers of kontekst: messages, tool calls, tool definitions, and typed IDs.
// Types live here because they are referenced by multiple packages that
// cannot import each other without creating dependency cycles.
package core
```

---

## 3. Interface Placement

**Reference:** interfaces.md — "Interfaces belong in consumer package"

### Interfaces to move to `agent` (the consumer)

All four interfaces are defined in implementer packages but consumed exclusively
by `internal/agent`:

| Interface | Current Location | Consumer |
|-----------|-----------------|----------|
| `Window` (was ContextWindow) | `conversation` | `agent.Agent` field |
| `Service` (was ContextService) | `conversation` | `agent.DefaultRunner` field |
| `Provider` | `provider` | `agent.Agent` field |
| `Creator` (was SessionCreator) | `session` | `agent.DefaultRunner` field |

**Strategy:** Export concrete types from the implementer packages. Define narrow
interfaces in `agent` where the consumer needs them (primarily for testability).

After this change:
- `conversation` exports `*FileService` (concrete) and `*Window` (concrete, was unexported `contextWindow`)
- `provider` exports `*OpenAI` (concrete, was `*OpenAIProvider`)
- `session` exports `*FileService` (concrete)
- `agent` defines the interfaces it needs:

```go
// agent/interfaces.go

// ConversationFactory creates conversation windows for agent runs.
type ConversationFactory interface {
    NewWindow(sessionID core.SessionID) (ConversationWindow, error)
}

// ConversationWindow manages the conversation context for a single agent run.
type ConversationWindow interface {
    SystemContent() string
    StartRun(params conversation.BudgetParams) error
    CompleteRun()
    AddMessage(msg core.Message) error
    BuildContext() ([]core.Message, error)
    SetAgentSystemPrompt(prompt string)
    SetActiveSkill(skill *core.SkillMetadata)
    ActiveSkill() *core.SkillMetadata
    Snapshot() conversation.Snapshot
}

// LLM generates chat completions from an LLM provider.
type LLM interface {
    GenerateChat(messages []core.Message, tools []core.ToolDef,
        sampling *core.SamplingConfig, model string, useToolRole bool) (provider.Response, error)
    CountTokens(text string) (int, error)
}

// SessionStore creates and ensures sessions exist on disk.
type SessionStore interface {
    Create() (core.SessionID, string, error)
    Ensure(sessionID core.SessionID) (string, error)
}
```

Note: `ConversationFactory.NewWindow()` returns the `ConversationWindow` interface
(defined in the same consumer package). The concrete `*conversation.FileService`
returns `*conversation.Window` which satisfies `ConversationWindow` implicitly.

Concrete types satisfy interfaces implicitly — no compile-time assertions needed
(though `var _ LLM = (*provider.OpenAI)(nil)` in tests is fine).

### Why `agent.Registry` stays in `agent`

The user noted that agent mixes orchestration with discovery. `agent.Registry`
is a config loader that scans directories for TOML files — different from the
agent loop.

However, `Registry.Load()` returns `*Agent` — it combines config loading with
agent construction (`agent.New()`). Extracting it to `config/agent` would require
either returning raw config (changing the API) or importing `agent` (creating a
cycle: `config/agent → agent → config/agent`).

**Decision:** Keep `Registry` in `agent`. It's a factory — "give me an agent by
name" — not just config discovery. Document this in the package comment.

### Dead interfaces to delete

| Interface | Location | Reason |
|-----------|----------|--------|
| `SessionMetadata` | `session` | Never consumed polymorphically; CLI uses `*FileService` directly |
| `SessionBrowser` | `session` | Same |
| `SessionService` | `session` | Composite of dead sub-interfaces |

Only `SessionCreator` (→ `agent.SessionStore`) is consumed polymorphically.

### Interfaces that stay where they are

| Interface | Location | Why |
|-----------|----------|-----|
| `Tool` | `tool` | `tool.Registry` (in same package) is the consumer. `builtin` provides implementations. Correct placement. |
| `Previewer` | `tool` | Optional capability check used by `tool.Registry.Preview()`. Correct. |
| `ToolExecutor` | `tool` | Consumed by `agent`, but both `tool.Registry` and `protocol.ToolExecutor` implement it. Can't move to `agent` without moving to the package that both implementers import. `tool` is the neutral home. |
| `Runner` | `agent` | Consumed by `protocol`, but signature references `agent.RunConfig`, `agent.Command`, `agent.Event`. Moving to `protocol` would create a cycle. Pinned in `agent`. |

---

## 4. Extract `internal/app/` — Server Entry Point

**Reference:** go-cobra project-layout.md, dependency-wiring.md

### Why a new package

`cli/serve.go` currently contains the full TCP server lifecycle: listener setup,
accept loop, connection handling, graceful shutdown, PID management, llama-server
subprocess management, and all service wiring. A newcomer asking "where does the
server run?" has to dig through a package named `cli`.

`internal/app/` makes the server entry point discoverable and separates server
concerns from CLI command parsing.

### What moves to `internal/app/`

```go
// internal/app/server.go

// Server is the kontekst daemon. It accepts ACP connections over TCP,
// dispatching each to a protocol handler backed by the agent runtime.
type Server struct {
    runner  *agent.DefaultRunner
    agents  *agent.Registry
    skills  *skill.Registry
    // ... all services
}

// NewServer wires all services from the given config.
func NewServer(cfg config.Config) (*Server, error) {
    // what cli/setup.go:setupServices() does today
}

// Serve accepts connections on addr until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, addr string) error {
    // what cli/serve.go:runServer() does today
    // TCP listen, accept loop, graceful shutdown
}
```

**From `cli/serve.go`:**
- `setupServices()` → `app.NewServer()`
- `runServer()` → `app.Server.Serve()`
- `handleConnection()` → `app.Server.handleConnection()` (unexported)
- `writePIDFile()` → `app.WritePIDFile()` or unexported
- `maxAgentContextSize()` → unexported helper in app
- `startLlamaServer()` → `app.StartLlama()` or stays in cli (subprocess management)

**From `cli/setup.go`:**
- `setupResult` struct → becomes `app.Server` fields
- `setupServices()` → becomes `app.NewServer()`

**Resulting `cli/serve.go`:**
```go
func runServeCmd(cmd *cobra.Command, args []string) error {
    srv, err := app.NewServer(cfg)
    if err != nil {
        return fmt.Errorf("build server: %w", err)
    }
    return srv.Serve(cmd.Context(), addr)
}
```

Thin adapter — parse flags, build server, serve. No business logic.

### New dependency layer

```
0: core, config, skill, command, tool/diff, tool/hashline
1: config/agent, provider, session, tool
2: conversation, tool/builtin
3: agent
4: protocol
5: app → protocol, agent, config/*, conversation, session, skill, command, tool/*
6: cli → app (server), protocol (client-side), config, session
7: cmd/cli → cli
```

`app` is the wiring layer. `cli` uses `app` for server operations and `protocol`
directly for client operations. No cycles.

### CLI cleanup (within `cli/`)

**4A. Clean up `root.go`** — After server code moves to `internal/app/`,
`root.go` keeps `NewRootCommand()` + CLI utilities (`loadConfig`, `resolveServer`,
`dialServer`, `loadActiveSession`, `saveActiveSession`, `startServer`,
`alreadyRunning`, PID helpers). ~170 lines — reasonable for a single file.

Colocate PID helpers (`readPID`, `writePIDFile`, `findProcessPID`, `pidofCommand`)
from `ps.go` and `serve.go` into `root.go` so they live together.

**4B. Extract shared callback helpers** — `init.go:50-83` duplicates session
update parsing from `run.go:handleSessionUpdate`. Extract shared helpers that
both `runCmd` and `runInitCmd` use.

**4C. Context bugs:**

| File | Line | Fix |
|------|------|-----|
| `cli/root.go` | `dialServer` | Add `ctx context.Context` parameter, pass to `protocol.Dial` |
| `cli/init.go` | 87 | Use `ctx` (from `cmd.Context()`) in retry dial loop |

**4D. Cobra settings** — Add `SilenceUsage: true` to root command.

---

## 5. File Organization

**Reference:** file-organization.md — Standard File Structure Template

Every file should follow: package comment → imports → constants → vars →
**exported types → constructors → exported methods** → unexported helpers.

### Files where exports are buried behind unexported implementation

**Priority fixes (exported functions hidden):**

| File | Issue | Fix |
|------|-------|-----|
| `config/agent/defaults.go` | `EnsureDefaults()` (only export) at line 99, after 96 lines of unexported `bundledAgents` data | Move `EnsureDefaults` before `bundledAgents` |
| `config/command/defaults.go` | Same pattern | Same fix |
| `config/skill/defaults.go` | Same pattern | Same fix |
| `agent/registry.go:148-181` | `NotFoundError`, `ConfigError` (exported error types) after unexported `fileExists` | Move error types to top, after `Registry`/`Summary` |
| `tool/builtin/builtin.go:14-68` | 5 unexported helpers before the only export `RegisterAll` | Move `RegisterAll` first |

**Medium fixes (unexported interspersed with exports):**

| File | Issue | Fix |
|------|-------|-----|
| `tool/builtin/edit_file.go:25-39,89` | Unexported `edit`, `editPlan` between exports; `prepareEdits` before exported `Preview` | Move unexported types and methods after all exports |
| `tool/diff/diff.go:322,334,339` | `SplitLines`, `GenerateStructuredDiff`, `GenerateStructuredDiffWithHashes` buried after 200 lines of unexported | Group all exports near top |
| `protocol/server.go:27-46` | `sessionState` + `sendCommand` before `NewHandler` | Move `sessionState` to bottom |
| `protocol/jsonrpc.go:31-48` | Unexported wire types between `Connection` and `RPCError` | Move wire types to bottom |
| `session/file_sessions.go:17-32` | `sessionMeta` at top; `sessionDir`/`sessionPath` before exports | Move to bottom |
| `tool/tools.go:11-26` | `contextKey` + helpers before `Tool`/`Previewer`/`ToolExecutor` interfaces | Move interfaces first |
| `provider/openai.go:58-82` | `CountTokens` before `GenerateChat` (primary workflow) | Swap order |

**Low (constants misplaced):**

| File | Issue |
|------|-------|
| `conversation/file.go:133` | `const chunkSize` after methods — move to top |
| `protocol/types/tooling.go:32-40` | `toolKindMap` between exports — move to bottom |

### Package doc comments

Every package needs a `// Package foo ...` comment. Priority:
1. `core` — most imported (27+ files)
2. `protocol` — the protocol layer
3. `agent` — orchestration
4. `tool` — tool abstractions
5. All remaining: `config`, `provider`, `session`, `conversation`, `skill`, `command`

---

## 6. Error Handling

**Reference:** error-handling.md — "Use `%w` to wrap errors"

### Bare `return err` to wrap

| File | Lines | Context to add |
|------|-------|---------------|
| `cli/root.go` | 150, 155, 162 | `"create log dir: %w"`, `"open log file: %w"`, `"start server: %w"` |
| `cli/session.go` | 29 | `"list sessions: %w"` |
| `cli/agents.go` | 26 | `"list agents: %w"` |
| `cli/serve.go` | 149 | `"create data dir: %w"` |
| `config/config.go` | 80, 85, 89, 95, 100, 104 | Contextual wrapping per operation |
| `session/file_sessions.go` | 109 | `"write session metadata: %w"` |
| `conversation/file.go` | 31 | `"append message: %w"` |

### Meaningful discarded errors to handle

| File | Line | Fix |
|------|------|-----|
| `cli/run.go` | 120 | `_ = saveActiveSession(...)` → `slog.Warn` on failure |
| `cli/setup.go` | 33, 43 | `os.MkdirAll(...)` unchecked → `slog.Warn` |
| `session/file_sessions.go` | 91 | `_ = json.Unmarshal(...)` → `slog.Warn` |
| `provider/logger.go` | 116 | `_ = os.MkdirAll(...)` → `slog.Warn` |

### Intentional best-effort (leave as-is)

- `core/id.go:39` — `rand.Read` never errors
- `protocol/server.go:145,302,434` — notification errors non-fatal
- `protocol/executor.go:233` — terminal release is best-effort
- `provider/logger.go:120,127-128` — best-effort debug logging

---

## 7. go:embed

**Reference:** file-organization.md — "Text > 4 lines should use `//go:embed`"

### `config/agent/defaults.go` — 4 inline TOML configs

Follow the existing `internal/config/agent/prompts/` pattern:

```
internal/config/agent/configs/
├── default.toml
├── coder.toml
├── fantasy.toml
└── init.toml
```

Then `//go:embed configs/default.toml` etc. in `defaults.go`.

---

## 8. Concurrency

### Race condition in `sessionState.sendCommand()`

**`protocol/server.go` (was acp/server.go):36-46:**

The mutex is released before the channel send. Between unlock and send, another
goroutine could close the channel, causing a panic.

**Reference:** concurrency.md — "Share memory by communicating"

**Fix:** Restructure to use select with a done channel:

```go
func (s *sessionState) sendCommand(cmd agent.Command) bool {
    s.mu.RLock()
    ch := s.commandCh
    done := s.doneCh
    s.mu.RUnlock()

    if ch == nil {
        return false
    }

    select {
    case ch <- cmd:
        return true
    case <-done:
        return false
    }
}
```

Where `doneCh` is closed when the run ends, guaranteeing the select won't block
on a nil/closed `commandCh`.

### `context.Background()` in protocol handlers

**`protocol/server.go:150,439`:**
```go
_ = h.conn.Notify(context.Background(), MethodSessionUpdate, ...)
```

Called within handlers that have `ctx`. Use the handler's context.

---

## 9. Naming

### Boolean functions

| Location | Current | Fix |
|----------|---------|-----|
| `protocol/server.go:411` | `isAllowOutcome()` | `outcomeIsAllowed()` |
| `agent/approvals.go:109` | `allDecided()` | `areAllDecided()` |
| `agent/approvals.go:118` | `hasAnyDenied()` | `anyWasDenied()` |
| `tool/builtin/builtin.go:21` | `isSafeRelative()` | `isRelativePathSafe()` |

### Receiver consistency in agent package

- `agent.go`: `(agent *Agent)` — full word
- `runner.go`: `(runner *DefaultRunner)` — full word
- `registry.go`: `(r *Registry)` — single letter

Standardize to abbreviated form: `(a *Agent)`, `(r *DefaultRunner)`, `(r *Registry)`.

### Custom type for approval state

Replace `*bool` tri-state with documented enum:

```go
// ApprovalState represents the outcome of a tool call approval request.
type ApprovalState int8

const (
    // ApprovalPending means the user has not yet responded.
    ApprovalPending ApprovalState = iota
    // ApprovalGranted means the user approved the tool call.
    ApprovalGranted
    // ApprovalDenied means the user denied the tool call.
    ApprovalDenied
)
```

---

## 10. Miscellaneous

- `protocol/types/session.go`: `McpServer struct{}` is empty. Delete or add TODO.
- `protocol/types/message.go:49`: `Update any` — document expected shapes.

---

## Implementation Sequence

### Phase 1: Package Renames (3 commits)

**1a.** Singular names — rename all plural packages to singular.
One atomic commit. `go build ./...` validates.

**1b.** Semantic renames — `context` → `conversation`, `acp` → `protocol`.
Also create `protocol/types/` subpackage (move `types_*.go` with topic names).
One atomic commit.

**1c.** Stutter removal — rename types that now stutter with their package.
`conversation.Window`, `session.Creator`, `session.FileService`, etc.
Delete dead session interfaces. One atomic commit.

### Phase 2: Core Trimming (1 commit)

Move `ChatResponse`/`Usage` → `provider`.
Move `ContextSnapshot`/`MessageStats` → `conversation`.
Inline `IntFromAny`.
Add package doc to core.

### Phase 3: Interface Placement (1 commit)

Move interfaces to `agent` (consumer).
Export concrete types from `conversation`, `provider`, `session`.
Delete dead interfaces.

### Phase 4: CLI Cleanup (2 commits)

**4a.** Extract `internal/app/` (server wiring + lifecycle from serve.go/setup.go).
Colocate PID helpers in root.go. Extract shared ACP callback helpers.

**4b.** Fix context.Background() bugs. Add SilenceUsage.

### Phase 5: File Organization (1 commit)

Reorder all files to match the Standard File Structure Template:
exports first, unexported at bottom. Package doc comments.

### Phase 6: Error Handling (1 commit)

Wrap bare returns. Handle meaningful discarded errors.

### Phase 7: go:embed + Concurrency + Naming (1 commit)

Extract TOML configs to embedded files.
Fix race condition in sendCommand.
Fix boolean names, receiver consistency, ApprovalState type.

---

## Verification

After each phase:

```bash
gofmt -l .                    # Must return empty
go build ./...                # Must succeed
go test ./...                 # All tests must pass
```

After all phases, smoke test:

```bash
make build
./bin/kontekst --help
./bin/kontekst serve --help
```

---

## Summary

| Phase | Commits | Risk | Touches |
|-------|---------|------|---------|
| 1: Renames | 3 | Medium (many files, but mechanical) | ~60 files |
| 2: Core trim | 1 | Low (cycle-verified moves) | ~15 files |
| 3: Interfaces | 1 | Medium (API surface change) | ~10 files |
| 4: CLI cleanup | 2 | Low (internal reorganization) | ~8 files |
| 5: File organization | 1 | None (reorder within files) | ~20 files |
| 6: Error handling | 1 | Low (behavior improvement) | ~10 files |
| 7: Embed + concurrency + naming | 1 | Low | ~15 files |

**Total: ~10 commits touching ~60 unique files.**
