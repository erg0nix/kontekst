package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/builtin"
)

type toolExecution struct {
	call   *pendingCall
	output string
	err    error
}

func (agent *Agent) executeTools(runID core.RunID, batchID string, calls []*pendingCall, eventChannel chan<- AgentEvent) error {
	skillCallbacks := &builtin.SkillCallbacks{
		ContextInjector: func(msg core.Message) error {
			return agent.context.AddMessage(msg)
		},
		SetActiveSkill: func(skill *core.SkillMetadata) {
			agent.context.SetActiveSkill(skill)
		},
	}

	var batch []toolExecution

	for _, call := range calls {
		exec := agent.executeToolCall(runID, call, skillCallbacks, eventChannel)
		batch = append(batch, exec)
	}

	if err := agent.addToolResults(batch); err != nil {
		return err
	}

	eventChannel <- AgentEvent{Type: EvtToolBatchCompleted, RunID: runID, BatchID: batchID}
	return nil
}

func (agent *Agent) executeToolCall(runID core.RunID, call *pendingCall, callbacks *builtin.SkillCallbacks, eventChannel chan<- AgentEvent) toolExecution {
	if call.Approved != nil && !*call.Approved {
		reason := call.Reason
		if reason == "" {
			reason = "user denied"
		}
		return toolExecution{call: call, err: fmt.Errorf("denied: %s", reason)}
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

	return toolExecution{call: call, output: output, err: err}
}

func (agent *Agent) addToolResults(batch []toolExecution) error {
	if len(batch) == 0 {
		return nil
	}

	if len(batch) == 1 {
		return agent.addSingleToolResult(batch[0])
	}

	return agent.addBatchedToolResults(batch)
}

func (agent *Agent) addSingleToolResult(exec toolExecution) error {
	var result core.ToolResult
	if exec.err != nil {
		result = core.ToolResult{
			CallID:  exec.call.ID,
			Name:    exec.call.Name,
			Output:  exec.err.Error(),
			IsError: true,
		}
	} else {
		result = core.ToolResult{
			CallID:  exec.call.ID,
			Name:    exec.call.Name,
			Output:  exec.output,
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

	return agent.context.AddMessage(msg)
}

func (agent *Agent) addBatchedToolResults(batch []toolExecution) error {
	var parts []string
	var callIDs []string
	hasError := false

	for _, exec := range batch {
		callIDs = append(callIDs, exec.call.ID)
		status := "SUCCESS"
		output := exec.output

		if exec.err != nil {
			status = "ERROR"
			output = exec.err.Error()
			hasError = true
		}

		parts = append(parts, fmt.Sprintf("[%s] %s\nStatus: %s\n%s",
			exec.call.ID, exec.call.Name, status, output))
	}

	mergedOutput := strings.Join(parts, "\n\n---\n\n")
	tokens, _ := agent.provider.CountTokens(mergedOutput)

	msg := core.Message{
		Role:    core.RoleTool,
		Content: mergedOutput,
		ToolResult: &core.ToolResult{
			CallID:  strings.Join(callIDs, ","),
			Name:    "batch_tool_results",
			Output:  mergedOutput,
			IsError: hasError,
		},
		Tokens: tokens,
	}

	return agent.context.AddMessage(msg)
}
