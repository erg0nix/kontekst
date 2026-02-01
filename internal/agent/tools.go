package agent

import (
	"context"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/builtin"
)

func (agent *Agent) executeTools(runID core.RunID, batchID string, calls []*pendingCall, eventChannel chan<- AgentEvent) error {
	skillCallbacks := &builtin.SkillCallbacks{
		ContextInjector: func(msg core.Message) error {
			return agent.context.AddMessage(msg)
		},
		SetActiveSkill: func(skill *core.SkillMetadata) {
			agent.context.SetActiveSkill(skill)
		},
	}

	for _, call := range calls {
		if call.Approved != nil && !*call.Approved {
			reason := call.Reason

			if reason == "" {
				reason = "denied"
			}

			result := core.ToolResult{CallID: call.ID, Name: call.Name, Output: "denied: " + reason, IsError: true}
			_ = agent.context.AddToolResult(result)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		if agent.workingDir != "" {
			ctx = tools.WithWorkingDir(ctx, agent.workingDir)
		}
		ctx = builtin.WithSkillCallbacks(ctx, skillCallbacks)
		eventChannel <- AgentEvent{Type: EvtToolStarted, RunID: runID, CallID: call.ID}
		output, err := agent.tools.Execute(call.Name, call.Args, ctx)
		cancel()

		if err != nil {
			result := core.ToolResult{CallID: call.ID, Name: call.Name, Output: err.Error(), IsError: true}
			_ = agent.context.AddToolResult(result)
			eventChannel <- AgentEvent{Type: EvtToolFailed, RunID: runID, CallID: call.ID, Error: err.Error()}
			continue
		}

		result := core.ToolResult{CallID: call.ID, Name: call.Name, Output: output, IsError: false}
		_ = agent.context.AddToolResult(result)
		eventChannel <- AgentEvent{Type: EvtToolCompleted, RunID: runID, CallID: call.ID, Output: output}
	}

	eventChannel <- AgentEvent{Type: EvtToolBatchCompleted, RunID: runID, BatchID: batchID}
	return nil
}
