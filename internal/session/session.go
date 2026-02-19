package session

import (
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

// Info holds metadata about a session including its size, message count, and timestamps.
type Info struct {
	ID           core.SessionID
	DefaultAgent string
	MessageCount int
	FileSize     int64
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

// Creator creates and ensures session existence.
type Creator interface {
	Create() (core.SessionID, string, error)
	Ensure(sessionID core.SessionID) (string, error)
}
