package acp

const ProtocolVersion = 1

const (
	MethodInitialize        = "initialize"
	MethodAuthenticate      = "authenticate"
	MethodSessionNew        = "session/new"
	MethodSessionLoad       = "session/load"
	MethodSessionPrompt     = "session/prompt"
	MethodSessionCancel     = "session/cancel"
	MethodSessionSetMode    = "session/set_mode"
	MethodSessionSetConfig  = "session/set_config_option"
	MethodSessionUpdate     = "session/update"
	MethodRequestPermission = "session/request_permission"
	MethodKontekstStatus    = "_kontekst/status"
	MethodKontekstShutdown  = "_kontekst/shutdown"
	MethodKontekstContext   = "_kontekst/context"

	MethodFsReadTextFile  = "fs/read_text_file"
	MethodFsWriteTextFile = "fs/write_text_file"
	MethodTerminalCreate  = "terminal/create"
	MethodTerminalOutput  = "terminal/output"
	MethodTerminalWait    = "terminal/wait_for_exit"
	MethodTerminalKill    = "terminal/kill"
	MethodTerminalRelease = "terminal/release"
)

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
