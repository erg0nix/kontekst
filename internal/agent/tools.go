package agent

import (
	"context"
	"fmt"
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
		output, err := agent.executeToolCall(runID, call, skillCallbacks, eventChannel)

		var result core.ToolResult
		if err != nil {
			result = core.ToolResult{
				CallID:  call.ID,
				Name:    call.Name,
				Output:  err.Error(),
				IsError: true,
			}
		} else {
			result = core.ToolResult{
				CallID:  call.ID,
				Name:    call.Name,
				Output:  output,
				IsError: false,
			}
		}

		tokens, _ := agent.provider.CountTokens(result.Output)
		msg := core.Message{
			Role:       core.RoleTool,
			Content:    result.Output,
			ToolResult: &result,
			Tokens:     tokens,
		}

		if err := agent.context.AddMessage(msg); err != nil {
			return err
		}
	}

	eventChannel <- AgentEvent{Type: EvtToolBatchCompleted, RunID: runID, BatchID: batchID}
	return nil
}

func (agent *Agent) executeToolCall(runID core.RunID, call *pendingCall, callbacks *builtin.SkillCallbacks, eventChannel chan<- AgentEvent) (string, error) {
	if call.Approved != nil && !*call.Approved {
		reason := call.Reason
		if reason == "" {
			reason = "user denied"
		}
		return "", fmt.Errorf("denied: %s", reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if agent.config.WorkingDir != "" {
		ctx = tools.WithWorkingDir(ctx, agent.config.WorkingDir)
	}
	ctx = builtin.WithSkillCallbacks(ctx, callbacks)

	eventChannel <- AgentEvent{Type: EvtToolStarted, RunID: runID, CallID: call.ID}
	output, err := agent.tools.Execute(call.Name, call.Args, ctx)

	if err != nil {
		eventChannel <- AgentEvent{Type: EvtToolFailed, RunID: runID, CallID: call.ID, Error: err.Error()}
	} else {
		eventChannel <- AgentEvent{Type: EvtToolCompleted, RunID: runID, CallID: call.ID, Output: output}
	}

	return output, err
}
