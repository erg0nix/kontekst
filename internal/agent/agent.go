package agent

import (
	"context"

	ctx "github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/tools"
)

type Agent struct {
	provider providers.ProviderRouter
	tools    tools.ToolExecutor
	context  ctx.ContextWindow
	config   RunConfig
}

func New(
	provider providers.ProviderRouter,
	toolExecutor tools.ToolExecutor,
	contextWindow ctx.ContextWindow,
	cfg RunConfig,
) *Agent {
	return &Agent{
		provider: provider,
		tools:    toolExecutor,
		context:  contextWindow,
		config:   cfg,
	}
}

func (agent *Agent) Run(prompt string) (chan<- AgentCommand, <-chan AgentEvent) {
	commandChannel := make(chan AgentCommand, 16)
	eventChannel := make(chan AgentEvent, 32)

	go agent.loop(prompt, commandChannel, eventChannel)

	return commandChannel, eventChannel
}

func (agent *Agent) loop(prompt string, commandChannel <-chan AgentCommand, eventChannel chan<- AgentEvent) {
	runID := core.NewRunID()
	eventChannel <- AgentEvent{Type: EvtRunStarted, RunID: runID}

	if prompt != "" {
		userMessage, _ := agent.context.RenderUserMessage(prompt)
		_ = agent.context.AddMessage(core.Message{Role: core.RoleUser, Content: userMessage})
	}

	for {
		contextMessages, _ := agent.context.BuildContext(agent.provider.CountTokens)
		chatResponse, err := agent.provider.GenerateChat(
			contextMessages,
			agent.tools.ToolDefinitions(),
			agent.config.Sampling,
			agent.config.Model,
			agent.config.ToolRole,
		)

		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		eventChannel <- AgentEvent{Type: EvtTurnCompleted, RunID: runID, Response: chatResponse}

		if len(chatResponse.ToolCalls) == 0 {
			_ = agent.context.AddMessage(core.Message{Role: core.RoleAssistant, Content: chatResponse.Content, AgentName: agent.config.AgentName})
			eventChannel <- AgentEvent{Type: EvtRunCompleted, RunID: runID, Response: chatResponse}
			return
		}

		batchID := newID("batch")
		pendingToolCalls := buildPending(chatResponse.ToolCalls)

		assistantMessage := core.Message{
			Role:      core.RoleAssistant,
			Content:   chatResponse.Content,
			ToolCalls: pendingToolCalls.asToolCalls(),
			AgentName: agent.config.AgentName,
		}
		_ = agent.context.AddMessage(assistantMessage)

		previewCtx := tools.WithWorkingDir(context.Background(), agent.config.WorkingDir)
		proposedCalls := pendingToolCalls.asProposed(agent.tools.Preview, previewCtx)

		eventChannel <- AgentEvent{Type: EvtToolBatch, RunID: runID, BatchID: batchID, Calls: proposedCalls}
		toolDecisions, err := collectApprovals(commandChannel, pendingToolCalls, batchID)
		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunCancelled, RunID: runID}
			return
		}

		if err := agent.executeTools(runID, batchID, toolDecisions, eventChannel); err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}
	}
}
