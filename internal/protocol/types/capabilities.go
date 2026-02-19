// Package types defines the ACP protocol request, response, and notification types.
package types

// InitializeRequest is the client's handshake request containing protocol version and capabilities.
type InitializeRequest struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities,omitempty"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
}

// InitializeResponse is the server's handshake response containing protocol version, capabilities, and auth methods.
type InitializeResponse struct {
	ProtocolVersion   int               `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities,omitempty"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
	AuthMethods       []AuthMethod      `json:"authMethods"`
}

// Implementation describes a named ACP client or agent with an optional version.
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities declares which optional features the client supports.
type ClientCapabilities struct {
	Fs       *FileSystemCapability `json:"fs,omitempty"`
	Terminal bool                  `json:"terminal,omitempty"`
}

// FileSystemCapability declares which filesystem operations the client can perform.
type FileSystemCapability struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

// AgentCapabilities declares which optional features the agent server supports.
type AgentCapabilities struct {
	LoadSession         bool                 `json:"loadSession,omitempty"`
	PromptCapabilities  *PromptCapabilities  `json:"promptCapabilities,omitempty"`
	McpCapabilities     *McpCapabilities     `json:"mcpCapabilities,omitempty"`
	SessionCapabilities *SessionCapabilities `json:"sessionCapabilities,omitempty"`
}

// PromptCapabilities declares which content types the agent accepts in prompts.
type PromptCapabilities struct {
	Image           bool `json:"image,omitempty"`
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
}

// McpCapabilities declares which MCP transport protocols the agent supports.
type McpCapabilities struct {
	HTTP bool `json:"http,omitempty"`
	SSE  bool `json:"sse,omitempty"`
}

// SessionCapabilities declares which optional session features the agent supports.
type SessionCapabilities struct{}

// AuthMethod describes an available authentication method offered by the agent.
type AuthMethod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AuthenticateRequest is a client request to authenticate using a specific method.
type AuthenticateRequest struct {
	MethodID string `json:"methodId"`
}

// AuthenticateResponse is the server's response to a successful authentication request.
type AuthenticateResponse struct{}
