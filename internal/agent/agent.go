package agent

import (
	"context"
	"encoding/json"
	"log/slog"

	ctx "github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/tools"
)

type Agent struct {
	provider providers.Provider
	tools    tools.ToolExecutor
	context  ctx.ContextWindow
	config   RunConfig
}

func New(
	provider providers.Provider,
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

	systemContent := agent.context.SystemContent()
	systemTokens, _ := agent.provider.CountTokens(systemContent)

	toolJSON, _ := json.Marshal(agent.tools.ToolDefinitions())
	toolTokens, _ := agent.provider.CountTokens(string(toolJSON))

	var userMessage string
	var userPromptTokens int
	if prompt != "" {
		userMessage, _ = agent.context.RenderUserMessage(prompt)
		userPromptTokens, _ = agent.provider.CountTokens(userMessage)
	}

	if err := agent.context.StartRun(ctx.BudgetParams{
		ContextSize:      agent.config.ContextSize,
		SystemContent:    systemContent,
		SystemTokens:     systemTokens,
		ToolTokens:       toolTokens,
		UserPromptTokens: userPromptTokens,
	}); err != nil {
		slog.Warn("failed to start context run", "error", err)
	}
	defer agent.context.CompleteRun()

	if userMessage != "" {
		if err := agent.context.AddMessage(core.Message{Role: core.RoleUser, Content: userMessage, Tokens: userPromptTokens}); err != nil {
			slog.Warn("failed to add user message", "error", err)
		}
	}

	for {
		contextMessages, err := agent.context.BuildContext()
		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		chatResponse, err := agent.provider.GenerateChat(
			contextMessages,
			agent.tools.ToolDefinitions(),
			agent.config.Sampling,
			agent.config.ProviderModel,
			agent.config.ToolRole,
		)

		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		completionTokens := 0
		if chatResponse.Usage != nil {
			completionTokens = chatResponse.Usage.CompletionTokens
		}

		if len(chatResponse.ToolCalls) == 0 {
			if err := agent.context.AddMessage(core.Message{
				Role:      core.RoleAssistant,
				Content:   chatResponse.Content,
				AgentName: agent.config.AgentName,
				Tokens:    completionTokens,
			}); err != nil {
				slog.Warn("failed to add assistant message", "error", err)
			}
			snapshot := agent.context.Snapshot()
			eventChannel <- AgentEvent{Type: EvtTurnCompleted, RunID: runID, Response: chatResponse, Snapshot: &snapshot}
			eventChannel <- AgentEvent{Type: EvtRunCompleted, RunID: runID, Response: chatResponse}
			return
		}

		pendingToolCalls := buildPending(chatResponse.ToolCalls)

		assistantMessage := core.Message{
			Role:      core.RoleAssistant,
			Content:   chatResponse.Content,
			ToolCalls: pendingToolCalls.asToolCalls(),
			AgentName: agent.config.AgentName,
			Tokens:    completionTokens,
		}
		if err := agent.context.AddMessage(assistantMessage); err != nil {
			slog.Warn("failed to add assistant message with tool calls", "error", err)
		}
		snapshot := agent.context.Snapshot()
		eventChannel <- AgentEvent{Type: EvtTurnCompleted, RunID: runID, Response: chatResponse, Snapshot: &snapshot}

		previewCtx := tools.WithWorkingDir(context.Background(), agent.config.WorkingDir)
		proposedCalls := pendingToolCalls.asProposed(agent.tools.Preview, previewCtx)

		eventChannel <- AgentEvent{Type: EvtToolsProposed, RunID: runID, Calls: proposedCalls}
		toolDecisions, err := collectApprovals(commandChannel, pendingToolCalls)
		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunCancelled, RunID: runID}
			return
		}

		if err := agent.executeTools(runID, toolDecisions, eventChannel); err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		if hasAnyDenied(toolDecisions) {
			eventChannel <- AgentEvent{Type: EvtRunCompleted, RunID: runID}
			return
		}
	}
}
