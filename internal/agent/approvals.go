package agent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/erg0nix/kontekst/internal/core"
)

// ApprovalState represents the state of a tool call approval decision.
type ApprovalState int

const (
	ApprovalPending ApprovalState = iota
	ApprovalGranted
	ApprovalDenied
)

type pendingCall struct {
	ID       string
	Name     string
	Args     map[string]any
	Approval ApprovalState
	Reason   string
}

type pendingBatch struct {
	calls map[string]*pendingCall
}

func buildPending(calls []core.ToolCall) *pendingBatch {
	out := &pendingBatch{calls: make(map[string]*pendingCall)}

	for _, call := range calls {
		callID := call.ID

		if callID == "" {
			callID = string(core.NewToolCallID())
		}

		out.calls[callID] = &pendingCall{ID: callID, Name: call.Name, Args: call.Arguments}
	}

	return out
}

type previewFunc func(name string, args map[string]any, ctx context.Context) (string, error)

func (b *pendingBatch) asProposed(preview previewFunc, ctx context.Context) []ProposedToolCall {
	var out []ProposedToolCall

	for _, call := range b.calls {
		argsJSON, _ := jsonMarshal(call.Args)
		proposed := ProposedToolCall{CallID: call.ID, Name: call.Name, ArgumentsJSON: argsJSON}

		if preview != nil {
			if previewText, err := preview(call.Name, call.Args, ctx); err == nil {
				proposed.Preview = previewText
			}
		}

		out = append(out, proposed)
	}

	return out
}

func (b *pendingBatch) asToolCalls() []core.ToolCall {
	var out []core.ToolCall

	for _, call := range b.calls {
		out = append(out, core.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Args})
	}

	return out
}

func collectApprovals(commandChannel <-chan Command, batch *pendingBatch) ([]*pendingCall, error) {
	for {
		if areAllDecided(batch) {
			return collectDecisions(batch), nil
		}

		command, ok := <-commandChannel
		if !ok {
			return nil, errors.New("command channel closed")
		}

		switch command.Type {
		case CmdCancel:
			return nil, errors.New("cancelled")
		case CmdApproveTool:
			if call, ok := batch.calls[command.CallID]; ok && call.Approval == ApprovalPending {
				call.Approval = ApprovalGranted
			}
		case CmdDenyTool:
			if call, ok := batch.calls[command.CallID]; ok && call.Approval == ApprovalPending {
				call.Approval = ApprovalDenied
				call.Reason = command.Reason
			}
		}
	}
}

func collectDecisions(batch *pendingBatch) []*pendingCall {
	var out []*pendingCall

	for _, call := range batch.calls {
		out = append(out, call)
	}

	return out
}

func areAllDecided(batch *pendingBatch) bool {
	for _, call := range batch.calls {
		if call.Approval == ApprovalPending {
			return false
		}
	}
	return true
}

func anyWasDenied(calls []*pendingCall) bool {
	for _, call := range calls {
		if call.Approval == ApprovalDenied {
			return true
		}
	}
	return false
}

func jsonMarshal(v map[string]any) (string, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return "{}", err
	}

	return string(data), nil
}
