package types

// PermissionOption represents a single approval choice presented to the user for a tool call.
type PermissionOption struct {
	OptionID string               `json:"optionId"`
	Name     string               `json:"name"`
	Kind     PermissionOptionKind `json:"kind"`
}

// PermissionOptionKind classifies a permission option as allowing or rejecting a tool call.
type PermissionOptionKind string

const (
	// PermissionOptionKindAllowOnce permits the tool call for this invocation only.
	PermissionOptionKindAllowOnce PermissionOptionKind = "allow_once"
	// PermissionOptionKindAllowAlways permits the tool call and all future calls of this type.
	PermissionOptionKindAllowAlways PermissionOptionKind = "allow_always"
	// PermissionOptionKindRejectOnce denies the tool call for this invocation only.
	PermissionOptionKindRejectOnce PermissionOptionKind = "reject_once"
	// PermissionOptionKindRejectAlways denies the tool call and all future calls of this type.
	PermissionOptionKindRejectAlways PermissionOptionKind = "reject_always"
)

// IsAllow reports whether the permission option kind grants permission.
func (k PermissionOptionKind) IsAllow() bool {
	return k == PermissionOptionKindAllowOnce || k == PermissionOptionKindAllowAlways
}

// ToolCallDetail describes a tool call for permission requests, including its input and preview.
type ToolCallDetail struct {
	ToolCallID ToolCallID      `json:"toolCallId"`
	Title      *string         `json:"title,omitempty"`
	Kind       *ToolKind       `json:"kind,omitempty"`
	Status     *ToolCallStatus `json:"status,omitempty"`
	RawInput   any             `json:"rawInput,omitempty"`
	Preview    any             `json:"preview,omitempty"`
}

// RequestPermissionRequest is the server's request asking the client to approve a tool call.
type RequestPermissionRequest struct {
	SessionID SessionID          `json:"sessionId"`
	ToolCall  ToolCallDetail     `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// RequestPermissionResponse is the client's response to a permission request.
type RequestPermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// PermissionOutcome represents the user's decision on a permission request.
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// PermissionSelected creates a PermissionOutcome indicating the user selected the given option.
func PermissionSelected(optionID string) PermissionOutcome {
	return PermissionOutcome{Outcome: "selected", OptionID: optionID}
}

// PermissionCancelled creates a PermissionOutcome indicating the user cancelled the permission prompt.
func PermissionCancelled() PermissionOutcome {
	return PermissionOutcome{Outcome: "cancelled"}
}
