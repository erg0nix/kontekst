package protocol

// ProtocolVersion is the current ACP protocol version.
const ProtocolVersion = 1

// ACP JSON-RPC method names for protocol handshake, session lifecycle, and extensions.
const (
	// MethodInitialize is the method for the protocol handshake request.
	MethodInitialize = "initialize"
	// MethodAuthenticate is the method for authentication requests.
	MethodAuthenticate = "authenticate"
	// MethodSessionNew is the method for creating a new session.
	MethodSessionNew = "session/new"
	// MethodSessionLoad is the method for loading an existing session.
	MethodSessionLoad = "session/load"
	// MethodSessionPrompt is the method for sending a prompt to a session.
	MethodSessionPrompt = "session/prompt"
	// MethodSessionCancel is the method for cancelling an active prompt.
	MethodSessionCancel = "session/cancel"
	// MethodSessionSetMode is the method for changing a session's mode.
	MethodSessionSetMode = "session/set_mode"
	// MethodSessionSetConfig is the method for setting a session configuration option.
	MethodSessionSetConfig = "session/set_config_option"
	// MethodSessionUpdate is the notification method for streaming session updates to the client.
	MethodSessionUpdate = "session/update"
	// MethodRequestPermission is the method for requesting tool execution approval from the client.
	MethodRequestPermission = "session/request_permission"
	// MethodKontekstStatus is the extension method for querying server status.
	MethodKontekstStatus = "_kontekst/status"
	// MethodKontekstShutdown is the extension method for shutting down the server.
	MethodKontekstShutdown = "_kontekst/shutdown"
	// MethodKontekstContext is the extension method for sending context snapshots to the client.
	MethodKontekstContext = "_kontekst/context"

	// MethodFsReadTextFile is the method for reading a text file via the client filesystem.
	MethodFsReadTextFile = "fs/read_text_file"
	// MethodFsWriteTextFile is the method for writing a text file via the client filesystem.
	MethodFsWriteTextFile = "fs/write_text_file"
	// MethodTerminalCreate is the method for creating a terminal process on the client.
	MethodTerminalCreate = "terminal/create"
	// MethodTerminalOutput is the method for retrieving terminal output from the client.
	MethodTerminalOutput = "terminal/output"
	// MethodTerminalWait is the method for waiting until a terminal process exits.
	MethodTerminalWait = "terminal/wait_for_exit"
	// MethodTerminalKill is the method for killing a terminal process on the client.
	MethodTerminalKill = "terminal/kill"
	// MethodTerminalRelease is the method for releasing a terminal's resources on the client.
	MethodTerminalRelease = "terminal/release"
)

// ErrorCode represents a JSON-RPC 2.0 error code.
type ErrorCode int

const (
	// ErrParseError indicates the server received invalid JSON.
	ErrParseError ErrorCode = -32700
	// ErrInvalidRequest indicates the JSON is not a valid JSON-RPC request.
	ErrInvalidRequest ErrorCode = -32600
	// ErrMethodNotFound indicates the requested method does not exist.
	ErrMethodNotFound ErrorCode = -32601
	// ErrInvalidParams indicates invalid method parameters were supplied.
	ErrInvalidParams ErrorCode = -32602
	// ErrInternalError indicates an internal server error occurred.
	ErrInternalError ErrorCode = -32603
	// ErrAuthRequired indicates authentication is required for the requested operation.
	ErrAuthRequired ErrorCode = -32000
	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound ErrorCode = -32002
)
