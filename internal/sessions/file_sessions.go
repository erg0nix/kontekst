package sessions

import (
	"os"
	"path/filepath"

	"github.com/erg0nix/kontekst/internal/core"
)

type FileSessionService struct {
	BaseDir string
}

func (service *FileSessionService) Create() (core.SessionID, string, error) {
	sessionID := core.NewSessionID()
	sessionDir := filepath.Join(service.BaseDir, "sessions")

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", "", err
	}

	sessionPath := filepath.Join(sessionDir, string(sessionID)+".jsonl")
	file, err := os.OpenFile(sessionPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", "", err
	}

	file.Close()

	return sessionID, sessionPath, nil
}

func (service *FileSessionService) Ensure(sessionID core.SessionID) (string, error) {
	sessionDir := filepath.Join(service.BaseDir, "sessions")

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", err
	}

	sessionPath := filepath.Join(sessionDir, string(sessionID)+".jsonl")
	file, err := os.OpenFile(sessionPath, os.O_CREATE, 0o644)
	if err != nil {
		return "", err
	}

	file.Close()

	return sessionPath, nil
}
