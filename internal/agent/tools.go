package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/tool"
	"github.com/erg0nix/kontekst/internal/tool/builtin"
)

func (a *Agent) executeTools(runID core.RunID, calls []*pendingCall, eventChannel chan<- Event) error {
	skillCallbacks := &builtin.SkillCallbacks{
		ContextInjector: func(msg core.Message) error {
			return a.context.AddMessage(msg)
		},
		SetActiveSkill: func(skill *core.SkillMetadata) {
			a.context.SetActiveSkill(skill)
		},
	}

	for _, call := range calls {
		output, err := a.executeToolCall(runID, call, skillCallbacks, eventChannel)

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

		tokens, _ := a.provider.CountTokens(result.Output)
		msg := core.Message{
			Role:       core.RoleTool,
			Content:    result.Output,
			ToolResult: &result,
			Tokens:     tokens,
		}

		if err := a.context.AddMessage(msg); err != nil {
			return err
		}
	}

	eventChannel <- Event{Type: EvtToolsCompleted, RunID: runID}
	return nil
}

func (a *Agent) executeToolCall(runID core.RunID, call *pendingCall, callbacks *builtin.SkillCallbacks, eventChannel chan<- Event) (string, error) {
	if call.Approval == ApprovalDenied {
		reason := call.Reason
		if reason == "" {
			reason = "user denied"
		}
		return "", fmt.Errorf("denied: %s", reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if a.config.WorkingDir != "" {
		ctx = tool.WithWorkingDir(ctx, a.config.WorkingDir)
	}
	ctx = builtin.WithSkillCallbacks(ctx, callbacks)

	eventChannel <- Event{Type: EvtToolStarted, RunID: runID, CallID: call.ID}
	output, err := a.tools.Execute(call.Name, call.Args, ctx)

	if err != nil {
		eventChannel <- Event{Type: EvtToolFailed, RunID: runID, CallID: call.ID, Error: err.Error()}
	} else {
		eventChannel <- Event{Type: EvtToolCompleted, RunID: runID, CallID: call.ID, Output: output}
	}

	return output, err
}
