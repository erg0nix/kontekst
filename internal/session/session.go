// Package session manages session creation, persistence, and metadata on disk.
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
