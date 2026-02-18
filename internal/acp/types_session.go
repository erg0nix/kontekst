package acp

type SessionID string

type NewSessionRequest struct {
	Cwd        string         `json:"cwd"`
	McpServers []McpServer    `json:"mcpServers"`
	Meta       map[string]any `json:"_meta,omitempty"`
}

type McpServer struct{}

type NewSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

type LoadSessionRequest struct {
	SessionID  SessionID   `json:"sessionId"`
	Cwd        string      `json:"cwd"`
	McpServers []McpServer `json:"mcpServers"`
}

type LoadSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

type SetSessionModeRequest struct {
	SessionID SessionID `json:"sessionId"`
	ModeID    string    `json:"modeId"`
}

type SetSessionModeResponse struct{}

type SetSessionConfigOptionRequest struct {
	SessionID SessionID `json:"sessionId"`
	ConfigID  string    `json:"configId"`
	Value     string    `json:"value"`
}

type SessionConfigOption struct {
	ID          string                     `json:"id"`
	Label       string                     `json:"label"`
	Values      []SessionConfigOptionValue `json:"values"`
	SelectedID  string                     `json:"selectedId"`
	Description string                     `json:"description,omitempty"`
}

type SessionConfigOptionValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type SetSessionConfigOptionResponse struct {
	ConfigOptions []SessionConfigOption `json:"configOptions"`
}
