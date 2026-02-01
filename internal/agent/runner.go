package agent

import (
	"github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
)

type Runner interface {
	StartRun(
		prompt string,
		sessionID core.SessionID,
		agentName string,
		agentSystemPrompt string,
		sampling *core.SamplingConfig,
		model string,
		workingDir string,
		skill *skills.Skill,
		skillContent string,
	) (chan<- AgentCommand, <-chan AgentEvent, error)
}

type AgentRunner struct {
	Provider providers.ProviderRouter
	Tools    tools.ToolExecutor
	Context  context.ContextService
	Sessions sessions.SessionService
	Runs     sessions.RunService
}

func (runner *AgentRunner) StartRun(
	prompt string,
	sessionID core.SessionID,
	agentName string,
	agentSystemPrompt string,
	sampling *core.SamplingConfig,
	model string,
	workingDir string,
	skill *skills.Skill,
	skillContent string,
) (chan<- AgentCommand, <-chan AgentEvent, error) {
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

	if agentSystemPrompt != "" {
		ctxWindow.SetAgentSystemPrompt(agentSystemPrompt)
	}

	if skill != nil && skillContent != "" {
		_ = ctxWindow.AddMessage(core.Message{Role: core.RoleUser, Content: skillContent})
	}

	agentEngine := New(runner.Provider, runner.Tools, ctxWindow, agentName, sampling, model, workingDir)
	agentEngine.SetActiveSkill(skill)
	commandChannel, eventChannel := agentEngine.Run(prompt)

	outputChannel := make(chan AgentEvent, 32)

	go func() {
		for event := range eventChannel {
			if event.Type == EvtRunStarted {
				event.SessionID = sessionID
				event.AgentName = agentName
			}

			switch event.Type {
			case EvtRunStarted:
				_ = runner.Runs.StartRun(sessionID, event.RunID)
			case EvtRunCompleted:
				_ = runner.Runs.CompleteRun(event.RunID)
			case EvtRunCancelled:
				_ = runner.Runs.CancelRun(event.RunID)
			case EvtRunFailed:
				_ = runner.Runs.FailRun(event.RunID)
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
