package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/erg0nix/kontekst/internal/core"
)

type pendingCall struct {
	ID       string
	Name     string
	Args     map[string]any
	Approved *bool
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
			callID = newID("call")
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

func (b *pendingBatch) asProposedUnapproved() []ProposedToolCall {
	var out []ProposedToolCall

	for _, call := range b.calls {
		if call.Approved != nil {
			continue
		}
		argsJSON, _ := jsonMarshal(call.Args)
		out = append(out, ProposedToolCall{CallID: call.ID, Name: call.Name, ArgumentsJSON: argsJSON})
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

func collectApprovals(commandChannel <-chan AgentCommand, batch *pendingBatch, batchID string) ([]*pendingCall, error) {
	for {
		if allDecided(batch) {
			return collectDecisions(batch), nil
		}

		command, ok := <-commandChannel
		if !ok {
			return nil, errors.New("command channel closed")
		}

		switch command.Type {
		case CmdCancel:
			return nil, errors.New("cancelled")
		case CmdApproveAll:
			if command.BatchID == batchID {
				for _, call := range batch.calls {
					if call.Approved == nil {
						v := true
						call.Approved = &v
					}
				}
			}
		case CmdDenyAll:
			if command.BatchID == batchID {
				for _, call := range batch.calls {
					if call.Approved == nil {
						v := false
						call.Approved = &v
						call.Reason = command.Reason
					}
				}
			}
		case CmdApproveTool:
			if call, ok := batch.calls[command.CallID]; ok && call.Approved == nil {
				v := true
				call.Approved = &v
			}
		case CmdDenyTool:
			if call, ok := batch.calls[command.CallID]; ok && call.Approved == nil {
				v := false
				call.Approved = &v
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

func allDecided(batch *pendingBatch) bool {
	for _, call := range batch.calls {
		if call.Approved == nil {
			return false
		}
	}
	return true
}

func jsonMarshal(v map[string]any) (string, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return "{}", err
	}

	return string(data), nil
}

func newID(prefix string) string {
	buffer := make([]byte, 6)
	_, _ = rand.Read(buffer)
	seed := hex.EncodeToString(buffer)

	return prefix + "_" + seed
}
