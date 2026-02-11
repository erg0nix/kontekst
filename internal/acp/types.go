package acp

const ProtocolVersion = 1

const (
	MethodInitialize      = "initialize"
	MethodAuthenticate    = "authenticate"
	MethodSessionNew      = "session/new"
	MethodSessionLoad     = "session/load"
	MethodSessionPrompt   = "session/prompt"
	MethodSessionCancel   = "session/cancel"
	MethodSessionSetMode  = "session/set_mode"
	MethodSessionSetCfg   = "session/set_config_option"
	MethodSessionUpdate   = "session/update"
	MethodRequestPerm     = "session/request_permission"
	MethodKontekstStatus  = "_kontekst/status"
	MethodKontekstShutdown = "_kontekst/shutdown"
	MethodKontekstContext  = "_kontekst/context"
)

type InitializeRequest struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities,omitempty"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
}

type InitializeResponse struct {
	ProtocolVersion   int               `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities,omitempty"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
	AuthMethods       []AuthMethod      `json:"authMethods"`
}

type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

type ClientCapabilities struct {
	Fs       *FileSystemCapability `json:"fs,omitempty"`
	Terminal bool                  `json:"terminal,omitempty"`
}

type FileSystemCapability struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

type AgentCapabilities struct {
	LoadSession         bool                 `json:"loadSession,omitempty"`
	PromptCapabilities  *PromptCapabilities  `json:"promptCapabilities,omitempty"`
	McpCapabilities     *McpCapabilities     `json:"mcpCapabilities,omitempty"`
	SessionCapabilities *SessionCapabilities `json:"sessionCapabilities,omitempty"`
}

type PromptCapabilities struct {
	Image           bool `json:"image,omitempty"`
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
}

type McpCapabilities struct {
	HTTP bool `json:"http,omitempty"`
	SSE  bool `json:"sse,omitempty"`
}

type SessionCapabilities struct{}

type AuthMethod struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type AuthenticateRequest struct {
	MethodId string `json:"methodId"`
}

type AuthenticateResponse struct{}

type SessionId string

type NewSessionRequest struct {
	Cwd        string         `json:"cwd"`
	McpServers []McpServer    `json:"mcpServers"`
	Meta       map[string]any `json:"_meta,omitempty"`
}

type McpServer struct{}

type NewSessionResponse struct {
	SessionId SessionId `json:"sessionId"`
}

type LoadSessionRequest struct {
	SessionId SessionId `json:"sessionId"`
}

type LoadSessionResponse struct {
	SessionId SessionId `json:"sessionId"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

type PromptRequest struct {
	SessionId SessionId      `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

type StopReason string

const (
	StopReasonEndTurn         StopReason = "end_turn"
	StopReasonMaxTokens       StopReason = "max_tokens"
	StopReasonMaxTurnRequests StopReason = "max_turn_requests"
	StopReasonRefusal         StopReason = "refusal"
	StopReasonCancelled       StopReason = "cancelled"
)

type PromptResponse struct {
	StopReason StopReason `json:"stopReason"`
}

type CancelNotification struct {
	SessionId SessionId `json:"sessionId"`
}

type SetSessionModeRequest struct {
	SessionId SessionId `json:"sessionId"`
}

type SetSessionModeResponse struct{}

type SetSessionConfigOptionRequest struct {
	SessionId SessionId `json:"sessionId"`
	ConfigId  string    `json:"configId"`
	ValueId   string    `json:"valueId"`
}

type SetSessionConfigOptionResponse struct{}

type ToolCallId string

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

type SessionNotification struct {
	SessionId SessionId `json:"sessionId"`
	Update    any       `json:"update"`
}

func AgentMessageChunk(text string) map[string]any {
	return map[string]any{
		"sessionUpdate": "agent_message_chunk",
		"content":       map[string]any{"type": "text", "text": text},
	}
}

func AgentThoughtChunk(text string) map[string]any {
	return map[string]any{
		"sessionUpdate": "agent_thought_chunk",
		"content":       map[string]any{"type": "text", "text": text},
	}
}

func ToolCallStart(id ToolCallId, title string, kind ToolKind, locations []ToolCallLocation, rawInput any) map[string]any {
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

func ToolCallUpdate(id ToolCallId, status ToolCallStatus, content []ToolCallContent, rawOutput any) map[string]any {
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

type PermissionOption struct {
	OptionId string               `json:"optionId"`
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

type ToolCallDetail struct {
	ToolCallId ToolCallId      `json:"toolCallId"`
	Title      *string         `json:"title,omitempty"`
	Kind       *ToolKind       `json:"kind,omitempty"`
	Status     *ToolCallStatus `json:"status,omitempty"`
	RawInput   any             `json:"rawInput,omitempty"`
}

type RequestPermissionRequest struct {
	SessionId SessionId          `json:"sessionId"`
	ToolCall  ToolCallDetail     `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

type RequestPermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
	Selected  *SelectedOutcome `json:"selected,omitempty"`
	Cancelled *struct{}        `json:"cancelled,omitempty"`
}

type SelectedOutcome struct {
	OptionId string `json:"optionId"`
}

type ErrorCode int

const (
	ErrParseError     ErrorCode = -32700
	ErrInvalidRequest ErrorCode = -32600
	ErrMethodNotFound ErrorCode = -32601
	ErrInvalidParams  ErrorCode = -32602
	ErrInternalError  ErrorCode = -32603
	ErrAuthRequired   ErrorCode = -32000
	ErrNotFound       ErrorCode = -32002
)

type StatusResponse struct {
	Bind      string `json:"bind"`
	Uptime    string `json:"uptime"`
	StartedAt string `json:"startedAt"`
	DataDir   string `json:"dataDir"`
}
