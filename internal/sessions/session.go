package sessions

import (
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

// SessionInfo holds metadata about a session including its size, message count, and timestamps.
type SessionInfo struct {
	ID           core.SessionID
	DefaultAgent string
	MessageCount int
	FileSize     int64
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

// SessionCreator defines methods for creating and ensuring session existence.
type SessionCreator interface {
	Create() (core.SessionID, string, error)
	Ensure(sessionID core.SessionID) (string, error)
}

// SessionMetadata defines methods for reading and writing session metadata such as the default agent.
type SessionMetadata interface {
	GetDefaultAgent(sessionID core.SessionID) (string, error)
	SetDefaultAgent(sessionID core.SessionID, agentName string) error
}

// SessionBrowser defines methods for listing and inspecting sessions.
type SessionBrowser interface {
	List() ([]SessionInfo, error)
	Get(sessionID core.SessionID) (SessionInfo, error)
}

// SessionService combines session creation, metadata, browsing, and deletion into a single interface.
type SessionService interface {
	SessionCreator
	SessionMetadata
	SessionBrowser
	Delete(sessionID core.SessionID) error
}
