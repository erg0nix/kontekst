package sessions

import (
	"encoding/json"
	"fmt"
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

func (service *FileSessionService) sessionDir() string {
	return filepath.Join(service.BaseDir, "sessions")
}

func (service *FileSessionService) sessionPath(id core.SessionID) string {
	return filepath.Join(service.sessionDir(), string(id)+".jsonl")
}

func (service *FileSessionService) Create() (core.SessionID, string, error) {
	sessionID := core.NewSessionID()

	if err := os.MkdirAll(service.sessionDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("create sessions directory: %w", err)
	}

	path := service.sessionPath(sessionID)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", "", fmt.Errorf("create session file: %w", err)
	}
	file.Close()

	return sessionID, path, nil
}

func (service *FileSessionService) Ensure(sessionID core.SessionID) (string, error) {
	if err := os.MkdirAll(service.sessionDir(), 0o755); err != nil {
		return "", fmt.Errorf("create sessions directory: %w", err)
	}

	path := service.sessionPath(sessionID)
	file, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return "", fmt.Errorf("ensure session file: %w", err)
	}
	file.Close()

	return path, nil
}

func (service *FileSessionService) metaPath(sessionID core.SessionID) string {
	return filepath.Join(service.sessionDir(), string(sessionID)+".meta.json")
}

func (service *FileSessionService) GetDefaultAgent(sessionID core.SessionID) (string, error) {
	data, err := os.ReadFile(service.metaPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read session metadata: %w", err)
	}

	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("parse session metadata: %w", err)
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
		return fmt.Errorf("marshal session metadata: %w", err)
	}

	return os.WriteFile(metaPath, data, 0o644)
}
