package acp

type ToolCallID string

type ToolKind string

const (
	ToolKindRead       ToolKind = "read"
	ToolKindEdit       ToolKind = "edit"
	ToolKindDelete     ToolKind = "delete"
	ToolKindMove       ToolKind = "move"
	ToolKindSearch     ToolKind = "search"
	ToolKindExecute    ToolKind = "execute"
	ToolKindThink      ToolKind = "think"
	ToolKindFetch      ToolKind = "fetch"
	ToolKindSwitchMode ToolKind = "switch_mode"
	ToolKindOther      ToolKind = "other"
)

var toolKindMap = map[string]ToolKind{
	"read_file":   ToolKindRead,
	"list_files":  ToolKindSearch,
	"write_file":  ToolKindEdit,
	"edit_file":   ToolKindEdit,
	"web_fetch":   ToolKindFetch,
	"run_command": ToolKindExecute,
	"skill":       ToolKindOther,
}

func ToolKindFromName(toolName string) ToolKind {
	if kind, ok := toolKindMap[toolName]; ok {
		return kind
	}
	return ToolKindOther
}

type ToolCallStatus string

const (
	ToolCallStatusPending    ToolCallStatus = "pending"
	ToolCallStatusInProgress ToolCallStatus = "in_progress"
	ToolCallStatusCompleted  ToolCallStatus = "completed"
	ToolCallStatusFailed     ToolCallStatus = "failed"
)

type ToolCallLocation struct {
	Path string `json:"path"`
	Line *int   `json:"line,omitempty"`
}

type ToolCallContent struct {
	Type    string        `json:"type"`
	Content *ContentBlock `json:"content,omitempty"`
}

func TextToolContent(text string) ToolCallContent {
	block := TextBlock(text)
	return ToolCallContent{Type: "content", Content: &block}
}

func ToolCallStart(id ToolCallID, title string, kind ToolKind, locations []ToolCallLocation, rawInput any) map[string]any {
	m := map[string]any{
		"sessionUpdate": "tool_call",
		"toolCallId":    id,
		"title":         title,
		"kind":          kind,
		"status":        ToolCallStatusPending,
		"content":       []any{},
	}
	if locations != nil {
		m["locations"] = locations
	}
	if rawInput != nil {
		m["rawInput"] = rawInput
	}
	return m
}

func ToolCallUpdate(id ToolCallID, status ToolCallStatus, content []ToolCallContent, rawOutput any) map[string]any {
	m := map[string]any{
		"sessionUpdate": "tool_call_update",
		"toolCallId":    id,
		"status":        status,
	}
	if content != nil {
		m["content"] = content
	}
	if rawOutput != nil {
		m["rawOutput"] = rawOutput
	}
	return m
}

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func AvailableCommandsUpdate(commands []Command) map[string]any {
	return map[string]any{
		"sessionUpdate":     "available_commands_update",
		"availableCommands": commands,
	}
}
