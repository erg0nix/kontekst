package acp

// StatusResponse contains server status information returned by the _kontekst/status method.
type StatusResponse struct {
	Bind      string `json:"bind"`
	Uptime    string `json:"uptime"`
	StartedAt string `json:"startedAt"`
	DataDir   string `json:"dataDir"`
}

// ReadTextFileRequest is a request to read a text file via the client filesystem.
type ReadTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Line      *uint32   `json:"line,omitempty"`
	Limit     *uint32   `json:"limit,omitempty"`
}

// ReadTextFileResponse contains the text content read from a file.
type ReadTextFileResponse struct {
	Content string `json:"content"`
}

// WriteTextFileRequest is a request to write text content to a file via the client filesystem.
type WriteTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
}

// WriteTextFileResponse is the response after successfully writing a file.
type WriteTextFileResponse struct{}

// EnvVariable represents a name-value pair for an environment variable.
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreateTerminalRequest is a request to create and start a terminal process on the client.
type CreateTerminalRequest struct {
	SessionID       SessionID     `json:"sessionId"`
	Command         string        `json:"command"`
	Args            []string      `json:"args,omitempty"`
	Cwd             string        `json:"cwd,omitempty"`
	Env             []EnvVariable `json:"env,omitempty"`
	OutputByteLimit *uint64       `json:"outputByteLimit,omitempty"`
}

// CreateTerminalResponse contains the ID of the newly created terminal process.
type CreateTerminalResponse struct {
	TerminalID string `json:"terminalId"`
}

// TerminalOutputRequest is a request to retrieve the output of a terminal process.
type TerminalOutputRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

// TerminalExitStatus contains the exit code or signal of a terminated process.
type TerminalExitStatus struct {
	ExitCode *uint32 `json:"exitCode,omitempty"`
	Signal   *string `json:"signal,omitempty"`
}

// TerminalOutputResponse contains the captured output from a terminal process.
type TerminalOutputResponse struct {
	Output     string              `json:"output"`
	Truncated  bool                `json:"truncated"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
}

// WaitForExitRequest is a request to block until a terminal process exits.
type WaitForExitRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

// WaitForExitResponse contains the exit code or signal after a terminal process exits.
type WaitForExitResponse struct {
	ExitCode *uint32 `json:"exitCode,omitempty"`
	Signal   *string `json:"signal,omitempty"`
}

// KillTerminalRequest is a request to forcibly terminate a terminal process.
type KillTerminalRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

// KillTerminalResponse is the response after killing a terminal process.
type KillTerminalResponse struct{}

// ReleaseTerminalRequest is a request to release a terminal's resources after it has exited.
type ReleaseTerminalRequest struct {
	SessionID  SessionID `json:"sessionId"`
	TerminalID string    `json:"terminalId"`
}

// ReleaseTerminalResponse is the response after releasing a terminal's resources.
type ReleaseTerminalResponse struct{}
