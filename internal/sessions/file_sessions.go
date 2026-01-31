package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/erg0nix/kontekst/internal/core"
)

type sessionMeta struct {
	DefaultAgent string `json:"default_agent,omitempty"`
}

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

func (service *FileSessionService) metaPath(sessionID core.SessionID) string {
	return filepath.Join(service.BaseDir, "sessions", string(sessionID)+".meta.json")
}

func (service *FileSessionService) GetDefaultAgent(sessionID core.SessionID) (string, error) {
	data, err := os.ReadFile(service.metaPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}

	return meta.DefaultAgent, nil
}

func (service *FileSessionService) SetDefaultAgent(sessionID core.SessionID, agentName string) error {
	metaPath := service.metaPath(sessionID)

	var meta sessionMeta
	data, err := os.ReadFile(metaPath)
	if err == nil {
		_ = json.Unmarshal(data, &meta)
	}

	meta.DefaultAgent = agentName

	data, err = json.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o644)
}
