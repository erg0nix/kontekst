package agent

import (
	"context"

	ctx "github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
)

type Agent struct {
	provider    providers.ProviderRouter
	tools       tools.ToolExecutor
	context     ctx.ContextWindow
	agentName   string
	sampling    *core.SamplingConfig
	model       string
	workingDir  string
	activeSkill *skills.Skill
}

func (agent *Agent) SetActiveSkill(skill *skills.Skill) {
	agent.activeSkill = skill
}

func (agent *Agent) isToolAutoApproved(toolName string) bool {
	if agent.activeSkill == nil {
		return false
	}
	for _, allowed := range agent.activeSkill.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

func New(
	provider providers.ProviderRouter,
	toolExecutor tools.ToolExecutor,
	contextWindow ctx.ContextWindow,
	agentName string,
	sampling *core.SamplingConfig,
	model string,
	workingDir string,
) *Agent {
	return &Agent{
		provider:   provider,
		tools:      toolExecutor,
		context:    contextWindow,
		agentName:  agentName,
		sampling:   sampling,
		model:      model,
		workingDir: workingDir,
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
		chatResponse, err := agent.provider.GenerateChat(contextMessages, agent.tools.ToolDefinitions(), nil, nil, agent.sampling, agent.model)

		if err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}

		if len(chatResponse.ToolCalls) == 0 {
			_ = agent.context.AddMessage(core.Message{Role: core.RoleAssistant, Content: chatResponse.Content, AgentName: agent.agentName})
			eventChannel <- AgentEvent{Type: EvtRunCompleted, RunID: runID, Response: chatResponse}
			return
		}

		batchID := newID("batch")
		pendingToolCalls := buildPending(chatResponse.ToolCalls)

		for _, call := range pendingToolCalls.calls {
			if agent.isToolAutoApproved(call.Name) {
				approved := true
				call.Approved = &approved
			}
		}

		assistantMessage := core.Message{
			Role:      core.RoleAssistant,
			Content:   chatResponse.Content,
			ToolCalls: pendingToolCalls.asToolCalls(),
			AgentName: agent.agentName,
		}
		_ = agent.context.AddMessage(assistantMessage)

		previewCtx := tools.WithWorkingDir(context.Background(), agent.workingDir)
		proposedCalls := pendingToolCalls.asProposed(agent.tools.Preview, previewCtx)

		var toolDecisions []*pendingCall
		if allDecided(pendingToolCalls) {
			toolDecisions = collectDecisions(pendingToolCalls)
		} else {
			eventChannel <- AgentEvent{Type: EvtToolBatch, RunID: runID, BatchID: batchID, Calls: proposedCalls}
			var err error
			toolDecisions, err = collectApprovals(commandChannel, pendingToolCalls, batchID)
			if err != nil {
				eventChannel <- AgentEvent{Type: EvtRunCancelled, RunID: runID}
				return
			}
		}

		if err := agent.executeTools(runID, batchID, toolDecisions, eventChannel); err != nil {
			eventChannel <- AgentEvent{Type: EvtRunFailed, RunID: runID, Error: err.Error()}
			return
		}
	}
}
