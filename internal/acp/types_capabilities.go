package acp

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
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthenticateRequest struct {
	MethodID string `json:"methodId"`
}

type AuthenticateResponse struct{}
