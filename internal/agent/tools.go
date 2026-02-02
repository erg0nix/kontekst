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
			tokens, _ := agent.provider.CountTokens(result.Output)
			msg := core.Message{Role: core.RoleTool, Content: result.Output, ToolResult: &result, Tokens: tokens}
			_ = agent.context.AddMessage(msg)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		if agent.config.WorkingDir != "" {
			ctx = tools.WithWorkingDir(ctx, agent.config.WorkingDir)
		}
		ctx = builtin.WithSkillCallbacks(ctx, skillCallbacks)
		eventChannel <- AgentEvent{Type: EvtToolStarted, RunID: runID, CallID: call.ID}
		output, err := agent.tools.Execute(call.Name, call.Args, ctx)
		cancel()

		if err != nil {
			result := core.ToolResult{CallID: call.ID, Name: call.Name, Output: err.Error(), IsError: true}
			tokens, _ := agent.provider.CountTokens(result.Output)
			msg := core.Message{Role: core.RoleTool, Content: result.Output, ToolResult: &result, Tokens: tokens}
			_ = agent.context.AddMessage(msg)
			eventChannel <- AgentEvent{Type: EvtToolFailed, RunID: runID, CallID: call.ID, Error: err.Error()}
			continue
		}

		result := core.ToolResult{CallID: call.ID, Name: call.Name, Output: output, IsError: false}
		tokens, _ := agent.provider.CountTokens(output)
		msg := core.Message{Role: core.RoleTool, Content: output, ToolResult: &result, Tokens: tokens}
		_ = agent.context.AddMessage(msg)
		eventChannel <- AgentEvent{Type: EvtToolCompleted, RunID: runID, CallID: call.ID, Output: output}
	}

	eventChannel <- AgentEvent{Type: EvtToolBatchCompleted, RunID: runID, BatchID: batchID}
	return nil
}
