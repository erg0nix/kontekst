package core

// Role represents the sender of a message in a conversation.
type Role string

const (
	// RoleSystem is the role for system-level instructions.
	RoleSystem Role = "system"
	// RoleUser is the role for user-provided messages.
	RoleUser Role = "user"
	// RoleAssistant is the role for LLM-generated responses.
	RoleAssistant Role = "assistant"
	// RoleTool is the role for tool execution results.
	RoleTool Role = "tool"
)

// Message represents a single message in a conversation with role, content, and optional tool interactions.
type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
	AgentName  string      `json:"agent_name,omitempty"`
	Tokens     int         `json:"tokens,omitempty"`
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResult holds the output of an executed tool call.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error,omitempty"`
}

// ToolDef describes a tool's name, description, and parameter schema for the LLM.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// SamplingConfig holds optional LLM sampling parameters like temperature and top-p.
type SamplingConfig struct {
	Temperature   *float64
	TopP          *float64
	TopK          *int
	RepeatPenalty *float64
	MaxTokens     *int
}

// SkillMetadata holds the name and file path of a loaded skill template.
type SkillMetadata struct {
	Name string
	Path string
}
