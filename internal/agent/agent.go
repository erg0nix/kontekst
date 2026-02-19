// Package agent implements the iterative agent loop that sends prompts to an LLM,
// detects tool calls, handles approval, executes tools, and feeds results back.
package agent

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/erg0nix/kontekst/internal/conversation"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/tool"
)

// Agent orchestrates the iterative loop of prompting an LLM, proposing tool calls, and executing approved tools.
type Agent struct {
	provider LLM
	tools    tool.ToolExecutor
	context  ConversationWindow
	config   RunConfig
}

// New creates an Agent with the given LLM provider, tool executor, context window, and run configuration.
func New(
	provider LLM,
	toolExecutor tool.ToolExecutor,
	contextWindow ConversationWindow,
	cfg RunConfig,
) *Agent {
	return &Agent{
		provider: provider,
		tools:    toolExecutor,
		context:  contextWindow,
		config:   cfg,
	}
}

// Run starts the agent loop in a goroutine and returns a command channel for client input and an event channel for agent output.
func (a *Agent) Run(prompt string) (chan<- Command, <-chan Event) {
	commandChannel := make(chan Command, 16)
	eventChannel := make(chan Event, 32)

	go a.loop(prompt, commandChannel, eventChannel)

	return commandChannel, eventChannel
}

func (a *Agent) loop(prompt string, commandChannel <-chan Command, eventChannel chan<- Event) {
	runID := core.NewRunID()
	eventChannel <- Event{Type: EvtRunStarted, RunID: runID}

	systemContent := a.context.SystemContent()
	systemTokens, err := a.provider.CountTokens(systemContent)
	if err != nil {
		slog.Warn("failed to count system tokens", "error", err)
	}

	toolJSON, _ := json.Marshal(a.tools.ToolDefinitions())
	toolTokens, err := a.provider.CountTokens(string(toolJSON))
	if err != nil {
		slog.Warn("failed to count tool tokens", "error", err)
	}

	userMessage := prompt
	var userPromptTokens int
	if userMessage != "" {
		userPromptTokens, err = a.provider.CountTokens(userMessage)
		if err != nil {
			slog.Warn("failed to count user prompt tokens", "error", err)
		}
	}

	if err := a.context.StartRun(conversation.BudgetParams{
		ContextSize:      a.config.ContextSize,
		SystemContent:    systemContent,
		SystemTokens:     systemTokens,
		ToolTokens:       toolTokens,
		UserPromptTokens: userPromptTokens,
	}); err != nil {
		slog.Warn("failed to start context run", "error", err)
	}
	defer a.context.CompleteRun()

	if userMessage != "" {
		if err := a.context.AddMessage(core.Message{Role: core.RoleUser, Content: userMessage, Tokens: userPromptTokens}); err != nil {
			slog.Warn("failed to add user message", "error", err)
		}
	}

	for {
		contextMessages, err := a.context.BuildContext()
		if err != nil {
			eventChannel <- Event{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		chatResponse, err := a.provider.GenerateChat(
			contextMessages,
			a.tools.ToolDefinitions(),
			a.config.Sampling,
			a.config.ProviderModel,
			a.config.ToolRole,
		)

		if err != nil {
			eventChannel <- Event{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		completionTokens := 0
		if chatResponse.Usage != nil {
			completionTokens = chatResponse.Usage.CompletionTokens
		}

		if len(chatResponse.ToolCalls) == 0 {
			if err := a.context.AddMessage(core.Message{
				Role:      core.RoleAssistant,
				Content:   chatResponse.Content,
				AgentName: a.config.AgentName,
				Tokens:    completionTokens,
			}); err != nil {
				slog.Warn("failed to add assistant message", "error", err)
			}
			snapshot := a.context.Snapshot()
			eventChannel <- Event{Type: EvtTurnCompleted, RunID: runID, Response: chatResponse, Snapshot: &snapshot}
			eventChannel <- Event{Type: EvtRunCompleted, RunID: runID, Response: chatResponse}
			return
		}

		pendingToolCalls := buildPending(chatResponse.ToolCalls)

		assistantMessage := core.Message{
			Role:      core.RoleAssistant,
			Content:   chatResponse.Content,
			ToolCalls: pendingToolCalls.asToolCalls(),
			AgentName: a.config.AgentName,
			Tokens:    completionTokens,
		}
		if err := a.context.AddMessage(assistantMessage); err != nil {
			slog.Warn("failed to add assistant message with tool calls", "error", err)
		}
		snapshot := a.context.Snapshot()
		eventChannel <- Event{Type: EvtTurnCompleted, RunID: runID, Response: chatResponse, Snapshot: &snapshot}

		previewCtx := tool.WithWorkingDir(context.Background(), a.config.WorkingDir)
		proposedCalls := pendingToolCalls.asProposed(a.tools.Preview, previewCtx)

		eventChannel <- Event{Type: EvtToolsProposed, RunID: runID, Calls: proposedCalls}
		toolDecisions, err := collectApprovals(commandChannel, pendingToolCalls)
		if err != nil {
			eventChannel <- Event{Type: EvtRunCancelled, RunID: runID}
			return
		}

		if err := a.executeTools(runID, toolDecisions, eventChannel); err != nil {
			eventChannel <- Event{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		if anyWasDenied(toolDecisions) {
			eventChannel <- Event{Type: EvtRunCompleted, RunID: runID}
			return
		}
	}
}
