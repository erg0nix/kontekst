package grpc

import (
	"context"
	"io"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/core"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
)

type AgentHandler struct {
	Runner   agent.Runner
	Registry *agent.Registry
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
				_ = stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: "run already started"}}})
				continue
			}

			startCommand := commandPayload.Start
			agentName := startCommand.AgentName
			agentSystemPrompt := ""
			agentModel := ""
			var agentSampling *core.SamplingConfig

			if agentName != "" && h.Registry != nil {
				loadedAgent, err := h.Registry.Load(agentName)
				if err != nil {
					_ = stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: err.Error()}}})
					continue
				}
				agentSystemPrompt = loadedAgent.SystemPrompt
				agentSampling = loadedAgent.Sampling
				agentModel = loadedAgent.Model
			}

			commandChannelForRun, eventChannelForRun, err := h.Runner.StartRun(
				startCommand.Prompt,
				core.SessionID(startCommand.SessionId),
				agentName,
				agentSystemPrompt,
				agentSampling,
				agentModel,
				startCommand.WorkingDir,
			)
			if err != nil {
				_ = stream.Send(&pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: err.Error()}}})
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
			_ = stream.Send(convertEvent(event))
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
	case agent.EvtToolStarted:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolStarted{ToolStarted: &pb.ToolExecutionStartedEvent{CallId: event.CallID}}}
	case agent.EvtToolCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolCompleted{ToolCompleted: &pb.ToolExecutionCompletedEvent{CallId: event.CallID, Output: event.Output}}}
	case agent.EvtToolFailed:
		return &pb.RunEvent{Event: &pb.RunEvent_ToolFailed{ToolFailed: &pb.ToolExecutionFailedEvent{CallId: event.CallID, Error: event.Error}}}
	case agent.EvtToolBatchCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_BatchCompleted{BatchCompleted: &pb.ToolBatchCompletedEvent{BatchId: event.BatchID}}}
	case agent.EvtRunCompleted:
		return &pb.RunEvent{Event: &pb.RunEvent_Completed{Completed: &pb.RunCompletedEvent{RunId: string(event.RunID), Content: event.Response.Content}}}
	case agent.EvtRunCancelled:
		return &pb.RunEvent{Event: &pb.RunEvent_Cancelled{Cancelled: &pb.RunCancelledEvent{RunId: string(event.RunID)}}}
	case agent.EvtRunFailed:
		return &pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{RunId: string(event.RunID), Error: event.Error}}}
	default:
		return &pb.RunEvent{Event: &pb.RunEvent_Failed{Failed: &pb.RunFailedEvent{Error: "unknown event"}}}
	}
}
