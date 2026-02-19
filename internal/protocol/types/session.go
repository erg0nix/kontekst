package types

// SessionID is a unique identifier for an ACP session.
type SessionID string

// NewSessionRequest is a client request to create a new agent session.
type NewSessionRequest struct {
	Cwd        string         `json:"cwd"`
	McpServers []McpServer    `json:"mcpServers"`
	Meta       map[string]any `json:"_meta,omitempty"`
}

// McpServer represents an MCP server configuration attached to a session.
// TODO:: populate with MCP server fields (name, endpoint, auth) when MCP support lands.
type McpServer struct{}

// NewSessionResponse is the server's response containing the newly created session ID.
type NewSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

// LoadSessionRequest is a client request to resume an existing session by ID.
type LoadSessionRequest struct {
	SessionID  SessionID   `json:"sessionId"`
	Cwd        string      `json:"cwd"`
	McpServers []McpServer `json:"mcpServers"`
}

// LoadSessionResponse is the server's response confirming the loaded session.
type LoadSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

// SetSessionModeRequest is a client request to change a session's operating mode.
type SetSessionModeRequest struct {
	SessionID SessionID `json:"sessionId"`
	ModeID    string    `json:"modeId"`
}

// SetSessionModeResponse is the server's response to a mode change request.
type SetSessionModeResponse struct{}

// SetSessionConfigOptionRequest is a client request to update a session configuration option.
type SetSessionConfigOptionRequest struct {
	SessionID SessionID `json:"sessionId"`
	ConfigID  string    `json:"configId"`
	Value     string    `json:"value"`
}

// SessionConfigOption describes a configurable session option with its possible values.
type SessionConfigOption struct {
	ID          string                     `json:"id"`
	Label       string                     `json:"label"`
	Values      []SessionConfigOptionValue `json:"values"`
	SelectedID  string                     `json:"selectedId"`
	Description string                     `json:"description,omitempty"`
}

// SessionConfigOptionValue represents one selectable value for a session configuration option.
type SessionConfigOptionValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// SetSessionConfigOptionResponse is the server's response containing the updated config options.
type SetSessionConfigOptionResponse struct {
	ConfigOptions []SessionConfigOption `json:"configOptions"`
}
