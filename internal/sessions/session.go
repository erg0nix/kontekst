package sessions

import (
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

type SessionInfo struct {
	ID           core.SessionID
	DefaultAgent string
	MessageCount int
	FileSize     int64
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

type SessionCreator interface {
	Create() (core.SessionID, string, error)
	Ensure(sessionID core.SessionID) (string, error)
}

type SessionMetadata interface {
	GetDefaultAgent(sessionID core.SessionID) (string, error)
	SetDefaultAgent(sessionID core.SessionID, agentName string) error
}

type SessionBrowser interface {
	List() ([]SessionInfo, error)
	Get(sessionID core.SessionID) (SessionInfo, error)
}

type SessionService interface {
	SessionCreator
	SessionMetadata
	SessionBrowser
	Delete(sessionID core.SessionID) error
}
