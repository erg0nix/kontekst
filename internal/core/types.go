package core

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
	AgentName  string      `json:"agent_name,omitempty"`
	Tokens     int         `json:"tokens,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ToolResult struct {
	CallID  string `json:"call_id"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error,omitempty"`
}

type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	Content   string
	Reasoning string
	ToolCalls []ToolCall
	Usage     *Usage
}

type SamplingConfig struct {
	Temperature   *float64
	TopP          *float64
	TopK          *int
	RepeatPenalty *float64
	MaxTokens     *int
}

type SkillMetadata struct {
	Name string
	Path string
}
