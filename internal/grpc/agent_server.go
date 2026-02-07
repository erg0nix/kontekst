package grpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/core"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
	"github.com/erg0nix/kontekst/internal/skills"
)

type AgentHandler struct {
	Runner   agent.Runner
	Registry *agent.Registry
	Skills   *skills.Registry
	pb.UnimplementedAgentServiceServer
}

func (h *AgentHandler) Run(stream pb.AgentService_RunServer) error {
	var commandChannel chan<- agent.AgentCommand
	var eventChannel <-chan agent.AgentEvent

	for {
		runCommand, err := stream.Recv()

		if err == io.EOF {
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdCancel}
			}
			return nil
		}

		if err != nil {
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdCancel}
			}
			return err
		}

		switch commandPayload := runCommand.Command.(type) {
		case *pb.RunCommand_Start:
			if commandChannel != nil {
				if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: "run already started"}}}); err != nil {
					slog.Warn("failed to send error event", "error", err)
				}
				continue
			}

			startCommand := commandPayload.Start
			agentName := startCommand.AgentName
			agentSystemPrompt := ""
			agentModel := ""
			agentToolRole := false
			var agentSampling *core.SamplingConfig

			if agentName != "" && h.Registry != nil {
				loadedAgent, err := h.Registry.Load(agentName)
				if err != nil {
					if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: err.Error()}}}); err != nil {
						slog.Warn("failed to send error event", "error", err)
					}
					continue
				}
				agentSystemPrompt = loadedAgent.SystemPrompt
				agentSampling = loadedAgent.Sampling
				agentModel = loadedAgent.Model
				agentToolRole = loadedAgent.ToolRole
			}

			var skill *skills.Skill
			var skillContent string
			if startCommand.Skill != nil && startCommand.Skill.Name != "" {
				if h.Skills == nil {
					if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: "skills not available"}}}); err != nil {
						slog.Warn("failed to send error event", "error", err)
					}
					continue
				}
				loadedSkill, found := h.Skills.Get(startCommand.Skill.Name)
				if !found {
					if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: fmt.Sprintf("skill not found: %s", startCommand.Skill.Name)}}}); err != nil {
						slog.Warn("failed to send error event", "error", err)
					}
					continue
				}
				rendered, err := loadedSkill.Render(startCommand.Skill.Arguments)
				if err != nil {
					if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: fmt.Sprintf("failed to render skill: %v", err)}}}); err != nil {
						slog.Warn("failed to send error event", "error", err)
					}
					continue
				}
				skill = loadedSkill
				skillContent = rendered
			}

			commandChannelForRun, eventChannelForRun, err := h.Runner.StartRun(agent.RunConfig{
				Prompt:            startCommand.Prompt,
				SessionID:         core.SessionID(startCommand.SessionId),
				AgentName:         agentName,
				AgentSystemPrompt: agentSystemPrompt,
				Sampling:          agentSampling,
				Model:             agentModel,
				WorkingDir:        startCommand.WorkingDir,
				Skill:             skill,
				SkillContent:      skillContent,
				ToolRole:          agentToolRole,
			})
			if err != nil {
				if err := stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: err.Error()}}}); err != nil {
					slog.Warn("failed to send error event", "error", err)
				}
				continue
			}

			commandChannel = commandChannelForRun
			eventChannel = eventChannelForRun

			go forwardEvents(stream.Context(), stream, eventChannel)
		case *pb.RunCommand_ApproveTool:
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdApproveTool, CallID: commandPayload.ApproveTool.CallId}
			}
		case *pb.RunCommand_DenyTool:
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdDenyTool, CallID: commandPayload.DenyTool.CallId, Reason: commandPayload.DenyTool.Reason}
			}
		case *pb.RunCommand_ApproveAll:
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdApproveAll, BatchID: commandPayload.ApproveAll.BatchId}
			}
		case *pb.RunCommand_DenyAll:
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdDenyAll, BatchID: commandPayload.DenyAll.BatchId, Reason: commandPayload.DenyAll.Reason}
			}
		case *pb.RunCommand_Cancel:
			if commandChannel != nil {
				commandChannel <- agent.AgentCommand{Type: agent.CmdCancel}
			}
		}
	}
}

func forwardEvents(ctx context.Context, stream pb.AgentService_RunServer, eventChannel <-chan agent.AgentEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventChannel:
			if !ok {
				return
			}
			if err := stream.Send(convertEvent(event)); err != nil {
				slog.Warn("stream broken, stopping event forwarding", "error", err)
				return
			}
		}
	}
}

func convertEvent(event agent.AgentEvent) *pb.RunEvent {
	switch event.Type {
	case agent.EvtRunStarted:
		return &pb.RunEvent{Event: &pb.RunEvent_Started{Started: &pb.RunStartedEvent{RunId: string(event.RunID), SessionId: string(event.SessionID), AgentName: event.AgentName}}}
	case agent.EvtTokenDelta:
		return &pb.RunEvent{Event: &pb.RunEvent_Token{Token: &pb.TokenDeltaEvent{Text: event.Token}}}
	case agent.EvtReasoningDelta:
		return &pb.RunEvent{Event: &pb.RunEvent_Reasoning{Reasoning: &pb.ReasoningDeltaEvent{Text: event.Reasoning}}}
	case agent.EvtToolBatch:
		proposedCalls := make([]*pb.ProposedToolCall, 0, len(event.Calls))

		for _, call := range event.Calls {
			proposedCalls = append(proposedCalls, &pb.ProposedToolCall{
				CallId:        call.CallID,
				Name:          call.Name,
				ArgumentsJson: call.ArgumentsJSON,
				Preview:       call.Preview,
			})
		}

		return &pb.RunEvent{Event: &pb.RunEvent_BatchProposed{BatchProposed: &pb.ToolBatchProposedEvent{BatchId: event.BatchID, Calls: proposedCalls}}}
	case agent.EvtTurnCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_TurnCompleted{TurnCompleted: &pb.TurnCompletedEvent{Content: event.Response.Content, Reasoning: event.Response.Reasoning}}}
	case agent.EvtToolStarted:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolStarted{ToolStarted: &pb.ToolExecutionStartedEvent{CallId: event.CallID}}}
	case agent.EvtToolCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolCompleted{ToolCompleted: &pb.ToolExecutionCompletedEvent{CallId: event.CallID, Output: event.Output}}}
	case agent.EvtToolFailed:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolFailed{ToolFailed: &pb.ToolExecutionFailedEvent{CallId: event.CallID, Error: event.Error}}}
	case agent.EvtToolBatchCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_BatchCompleted{BatchCompleted: &pb.ToolBatchCompletedEvent{BatchId: event.BatchID}}}
	case agent.EvtRunCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_Completed{Completed: &pb.RunCompletedEvent{RunId: string(event.RunID), Content: event.Response.Content, Reasoning: event.Response.Reasoning}}}
	case agent.EvtRunCancelled:
		return &pb.RunEvent{Event: &pb.RunEvent_Cancelled{Cancelled: &pb.RunCancelledEvent{RunId: string(event.RunID)}}}
	case agent.EvtRunFailed:
		return &pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{RunId: string(event.RunID), Error: event.Error}}}
	default:
		return &pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: "unknown event"}}}
	}
}
