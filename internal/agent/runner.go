package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/provider"
	"github.com/erg0nix/kontekst/internal/session"
	"github.com/erg0nix/kontekst/internal/skill"
	"github.com/erg0nix/kontekst/internal/tool"
)

// RunConfig holds all parameters needed to start a single agent run.
type RunConfig struct {
	Prompt              string
	SessionID           core.SessionID
	AgentName           string
	AgentSystemPrompt   string
	ContextSize         int
	Sampling            *core.SamplingConfig
	ProviderEndpoint    string
	ProviderModel       string
	ProviderHTTPTimeout time.Duration
	WorkingDir          string
	Skill               *skill.Skill
	SkillContent        string
	ToolRole            bool
	Tools               tool.ToolExecutor
}

// Runner starts agent runs and returns channels for bidirectional communication.
type Runner interface {
	StartRun(cfg RunConfig) (chan<- Command, <-chan Event, error)
}

// DefaultRunner is the standard Runner implementation that wires together sessions, context, and an LLM provider.
type DefaultRunner struct {
	Tools       tool.ToolExecutor
	Context     context.ContextService
	Sessions    session.SessionCreator
	DebugConfig config.DebugConfig
}

// StartRun initializes a session and context window, then starts the agent loop in a background goroutine.
func (runner *DefaultRunner) StartRun(cfg RunConfig) (chan<- Command, <-chan Event, error) {
	sessionID := cfg.SessionID
	if sessionID == "" {
		newSessionID, _, err := runner.Sessions.Create()
		if err != nil {
			return nil, nil, err
		}

		sessionID = newSessionID
	} else {
		if _, err := runner.Sessions.Ensure(sessionID); err != nil {
			return nil, nil, err
		}
	}

	ctxWindow, err := runner.Context.NewWindow(sessionID)
	if err != nil {
		return nil, nil, err
	}

	if cfg.AgentSystemPrompt != "" {
		ctxWindow.SetAgentSystemPrompt(cfg.AgentSystemPrompt)
	}

	prompt := cfg.Prompt
	if cfg.Skill != nil && cfg.SkillContent != "" {
		ctxWindow.SetActiveSkill(&core.SkillMetadata{Name: cfg.Skill.Name, Path: cfg.Skill.Path})
		prompt = fmt.Sprintf("%s\n\n---\n\n%s", cfg.Skill.FormatContent(cfg.SkillContent), prompt)
	}

	if cfg.WorkingDir != "" {
		agentsMDPath := filepath.Join(cfg.WorkingDir, "AGENTS.md")
		content, err := os.ReadFile(agentsMDPath)
		if err == nil {
			prompt = fmt.Sprintf("<project-instructions>\n%s\n</project-instructions>\n\n%s",
				strings.TrimSpace(string(content)), prompt)
		} else if !os.IsNotExist(err) {
			slog.Warn("failed to read AGENTS.md", "path", agentsMDPath, "error", err)
		}
	}

	provider := provider.NewOpenAIProvider(
		provider.OpenAIConfig{
			Endpoint:    cfg.ProviderEndpoint,
			HTTPTimeout: cfg.ProviderHTTPTimeout,
		},
		runner.DebugConfig,
	)

	toolExecutor := cfg.Tools
	if toolExecutor == nil {
		toolExecutor = runner.Tools
	}

	agentEngine := New(provider, toolExecutor, ctxWindow, cfg)
	commandChannel, eventChannel := agentEngine.Run(prompt)

	outputChannel := make(chan Event, 32)

	go func() {
		turnCounter := 0
		for event := range eventChannel {
			if event.Type == EvtRunStarted {
				event.SessionID = sessionID
				event.AgentName = cfg.AgentName
			}

			switch event.Type {
			case EvtRunStarted:
				slog.Info("run started", "run_id", event.RunID, "session_id", sessionID)
			case EvtTurnCompleted:
				turnCounter++
				if event.Snapshot != nil {
					slog.Info("context snapshot",
						"run_id", event.RunID,
						"turn", turnCounter,
						"context_size", event.Snapshot.ContextSize,
						"total_tokens", event.Snapshot.TotalTokens,
						"remaining_tokens", event.Snapshot.RemainingTokens,
						"history_tokens", event.Snapshot.HistoryTokens,
						"history_messages", event.Snapshot.HistoryMessages,
						"total_messages", event.Snapshot.TotalMessages,
					)
				}
			case EvtRunCompleted:
				slog.Info("run completed", "run_id", event.RunID)
			case EvtRunCancelled:
				slog.Info("run cancelled", "run_id", event.RunID)
			case EvtRunFailed:
				slog.Info("run failed", "run_id", event.RunID)
			}

			outputChannel <- event

			if event.Type == EvtRunCompleted || event.Type == EvtRunCancelled || event.Type == EvtRunFailed {
				close(outputChannel)
				return
			}
		}
	}()

	return commandChannel, outputChannel, nil
}

var _ Runner = (*DefaultRunner)(nil)
