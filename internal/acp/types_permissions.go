package acp

type PermissionOption struct {
	OptionID string               `json:"optionId"`
	Name     string               `json:"name"`
	Kind     PermissionOptionKind `json:"kind"`
}

type PermissionOptionKind string

const (
	PermissionOptionKindAllowOnce    PermissionOptionKind = "allow_once"
	PermissionOptionKindAllowAlways  PermissionOptionKind = "allow_always"
	PermissionOptionKindRejectOnce   PermissionOptionKind = "reject_once"
	PermissionOptionKindRejectAlways PermissionOptionKind = "reject_always"
)

func (k PermissionOptionKind) IsAllow() bool {
	return k == PermissionOptionKindAllowOnce || k == PermissionOptionKindAllowAlways
}

type ToolCallDetail struct {
	ToolCallID ToolCallID      `json:"toolCallId"`
	Title      *string         `json:"title,omitempty"`
	Kind       *ToolKind       `json:"kind,omitempty"`
	Status     *ToolCallStatus `json:"status,omitempty"`
	RawInput   any             `json:"rawInput,omitempty"`
	Preview    any             `json:"preview,omitempty"`
}

type RequestPermissionRequest struct {
	SessionID SessionID          `json:"sessionId"`
	ToolCall  ToolCallDetail     `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

type RequestPermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

func PermissionSelected(optionID string) PermissionOutcome {
	return PermissionOutcome{Outcome: "selected", OptionID: optionID}
}

func PermissionCancelled() PermissionOutcome {
	return PermissionOutcome{Outcome: "cancelled"}
}
