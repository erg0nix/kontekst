package acp

type StatusResponse struct {
	Bind      string `json:"bind"`
	Uptime    string `json:"uptime"`
	StartedAt string `json:"startedAt"`
	DataDir   string `json:"dataDir"`
}

type ReadTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Line      *uint32   `json:"line,omitempty"`
	Limit     *uint32   `json:"limit,omitempty"`
}

type ReadTextFileResponse struct {
	Content string `json:"content"`
}

type WriteTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
}

type WriteTextFileResponse struct{}

type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CreateTerminalRequest struct {
	SessionID       SessionID     `json:"sessionId"`
	Command         string        `json:"command"`
	Args            []string      `json:"args,omitempty"`
	Cwd             string        `json:"cwd,omitempty"`
	Env             []EnvVariable `json:"env,omitempty"`
	OutputByteLimit *uint64       `json:"outputByteLimit,omitempty"`
}

type CreateTerminalResponse struct {
	TerminalID string `json:"terminalId"`
}

type TerminalOutputRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

type TerminalExitStatus struct {
	ExitCode *uint32 `json:"exitCode,omitempty"`
	Signal   *string `json:"signal,omitempty"`
}

type TerminalOutputResponse struct {
	Output     string              `json:"output"`
	Truncated  bool                `json:"truncated"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
}

type WaitForExitRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

type WaitForExitResponse struct {
	ExitCode *uint32 `json:"exitCode,omitempty"`
	Signal   *string `json:"signal,omitempty"`
}

type KillTerminalRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

type KillTerminalResponse struct{}

type ReleaseTerminalRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

type ReleaseTerminalResponse struct{}
