package acp

// ToolCallID is a unique identifier for a tool call within a session.
type ToolCallID string

// ToolKind categorizes a tool call for display purposes.
type ToolKind string

const (
	// ToolKindRead indicates a tool that reads data without modification.
	ToolKindRead ToolKind = "read"
	// ToolKindEdit indicates a tool that modifies existing content.
	ToolKindEdit ToolKind = "edit"
	// ToolKindDelete indicates a tool that removes content.
	ToolKindDelete ToolKind = "delete"
	// ToolKindMove indicates a tool that relocates content.
	ToolKindMove ToolKind = "move"
	// ToolKindSearch indicates a tool that searches or lists content.
	ToolKindSearch ToolKind = "search"
	// ToolKindExecute indicates a tool that runs a command or process.
	ToolKindExecute ToolKind = "execute"
	// ToolKindThink indicates a tool used for internal reasoning.
	ToolKindThink ToolKind = "think"
	// ToolKindFetch indicates a tool that fetches remote content.
	ToolKindFetch ToolKind = "fetch"
	// ToolKindSwitchMode indicates a tool that changes the agent's operating mode.
	ToolKindSwitchMode ToolKind = "switch_mode"
	// ToolKindOther indicates a tool that does not fit other categories.
	ToolKindOther ToolKind = "other"
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

// ToolKindFromName maps a tool name to its corresponding ToolKind.
func ToolKindFromName(toolName string) ToolKind {
	if kind, ok := toolKindMap[toolName]; ok {
		return kind
	}
	return ToolKindOther
}

// ToolCallStatus represents the current execution state of a tool call.
type ToolCallStatus string

const (
	// ToolCallStatusPending indicates the tool call is awaiting approval.
	ToolCallStatusPending ToolCallStatus = "pending"
	// ToolCallStatusInProgress indicates the tool call is currently executing.
	ToolCallStatusInProgress ToolCallStatus = "in_progress"
	// ToolCallStatusCompleted indicates the tool call finished successfully.
	ToolCallStatusCompleted ToolCallStatus = "completed"
	// ToolCallStatusFailed indicates the tool call encountered an error.
	ToolCallStatusFailed ToolCallStatus = "failed"
)

// ToolCallLocation represents a file path and optional line number associated with a tool call.
type ToolCallLocation struct {
	Path string `json:"path"`
	Line *int   `json:"line,omitempty"`
}

// ToolCallContent represents a piece of content produced by a tool call.
type ToolCallContent struct {
	Type    string        `json:"type"`
	Content *ContentBlock `json:"content,omitempty"`
}

// TextToolContent creates a ToolCallContent wrapping a text content block.
func TextToolContent(text string) ToolCallContent {
	block := TextBlock(text)
	return ToolCallContent{Type: "content", Content: &block}
}

// ToolCallStart creates a session update payload announcing a new tool call.
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

// ToolCallUpdate creates a session update payload with a tool call's new status and output.
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

// Command represents a slash command available to the user within a session.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AvailableCommandsUpdate creates a session update payload listing available slash commands.
func AvailableCommandsUpdate(commands []Command) map[string]any {
	return map[string]any{
		"sessionUpdate":     "available_commands_update",
		"availableCommands": commands,
	}
}
