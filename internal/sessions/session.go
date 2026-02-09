package sessions

import "github.com/erg0nix/kontekst/internal/core"

type SessionService interface {
	Create() (core.SessionID, string, error)
	Ensure(sessionID core.SessionID) (string, error)
	GetDefaultAgent(sessionID core.SessionID) (string, error)
	SetDefaultAgent(sessionID core.SessionID, agentName string) error
}
