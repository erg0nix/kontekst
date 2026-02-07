package agent

import (
	"fmt"
	"log/slog"

	"github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
)

type RunConfig struct {
	Prompt            string
	SessionID         core.SessionID
	AgentName         string
	AgentSystemPrompt string
	Sampling          *core.SamplingConfig
	Model             string
	WorkingDir        string
	Skill             *skills.Skill
	SkillContent      string
	ToolRole          bool
}

type Runner interface {
	StartRun(cfg RunConfig) (chan<- AgentCommand, <-chan AgentEvent, error)
}

type AgentRunner struct {
	Provider providers.ProviderRouter
	Tools    tools.ToolExecutor
	Context  context.ContextService
	Sessions sessions.SessionService
	Runs     sessions.RunService
}

func (runner *AgentRunner) StartRun(cfg RunConfig) (chan<- AgentCommand, <-chan AgentEvent, error) {
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

	agentEngine := New(runner.Provider, runner.Tools, ctxWindow, cfg)
	commandChannel, eventChannel := agentEngine.Run(prompt)

	outputChannel := make(chan AgentEvent, 32)

	go func() {
		for event := range eventChannel {
			if event.Type == EvtRunStarted {
				event.SessionID = sessionID
				event.AgentName = cfg.AgentName
			}

			switch event.Type {
			case EvtRunStarted:
				if err := runner.Runs.StartRun(sessionID, event.RunID); err != nil {
					slog.Warn("failed to record run start", "run_id", event.RunID, "error", err)
				}
			case EvtRunCompleted:
				if err := runner.Runs.CompleteRun(event.RunID); err != nil {
					slog.Warn("failed to record run completion", "run_id", event.RunID, "error", err)
				}
			case EvtRunCancelled:
				if err := runner.Runs.CancelRun(event.RunID); err != nil {
					slog.Warn("failed to record run cancellation", "run_id", event.RunID, "error", err)
				}
			case EvtRunFailed:
				if err := runner.Runs.FailRun(event.RunID); err != nil {
					slog.Warn("failed to record run failure", "run_id", event.RunID, "error", err)
				}
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

var _ Runner = (*AgentRunner)(nil)
