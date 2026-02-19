package session

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

// FileService implements Service using JSONL files on the local filesystem.
type FileService struct {
	BaseDir string
}

func (service *FileService) sessionDir() string {
	return filepath.Join(service.BaseDir, "sessions")
}

func (service *FileService) sessionPath(id core.SessionID) string {
	return filepath.Join(service.sessionDir(), string(id)+".jsonl")
}

// Create generates a new session ID and creates its backing file, returning the ID and file path.
func (service *FileService) Create() (core.SessionID, string, error) {
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

// Ensure creates the session file if it does not already exist and returns its path.
func (service *FileService) Ensure(sessionID core.SessionID) (string, error) {
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

func (service *FileService) metaPath(sessionID core.SessionID) string {
	return filepath.Join(service.sessionDir(), string(sessionID)+".meta.json")
}

// GetDefaultAgent returns the default agent name stored in the session's metadata file.
func (service *FileService) GetDefaultAgent(sessionID core.SessionID) (string, error) {
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

// SetDefaultAgent persists the given agent name as the default for the session.
func (service *FileService) SetDefaultAgent(sessionID core.SessionID, agentName string) error {
	metaPath := service.metaPath(sessionID)

	var meta sessionMeta
	data, err := os.ReadFile(metaPath)
	if err == nil {
		if err := json.Unmarshal(data, &meta); err != nil {
			slog.Warn("failed to parse session metadata", "path", metaPath, "error", err)
		}
	}

	meta.DefaultAgent = agentName

	data, err = json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal session metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("write session metadata: %w", err)
	}

	return nil
}

// List returns all sessions sorted by most recently modified first.
func (service *FileService) List() ([]Info, error) {
	dir := service.sessionDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var result []Info
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		id := core.SessionID(strings.TrimSuffix(entry.Name(), ".jsonl"))
		info, err := service.Get(id)
		if err != nil {
			continue
		}
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ModifiedAt.After(result[j].ModifiedAt)
	})

	return result, nil
}

// Get returns metadata for a single session identified by its ID.
func (service *FileService) Get(sessionID core.SessionID) (Info, error) {
	path := service.sessionPath(sessionID)

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Info{}, fmt.Errorf("session not found: %s", sessionID)
		}
		return Info{}, fmt.Errorf("stat session: %w", err)
	}

	agent, _ := service.GetDefaultAgent(sessionID)

	return Info{
		ID:           sessionID,
		DefaultAgent: agent,
		MessageCount: countLines(path),
		FileSize:     stat.Size(),
		CreatedAt:    parseSessionTimestamp(sessionID),
		ModifiedAt:   stat.ModTime(),
	}, nil
}

// Delete removes the session's data and metadata files from disk.
func (service *FileService) Delete(sessionID core.SessionID) error {
	path := service.sessionPath(sessionID)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		return fmt.Errorf("stat session: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	metaPath := service.metaPath(sessionID)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete session metadata: %w", err)
	}

	return nil
}

type sessionMeta struct {
	DefaultAgent string `json:"default_agent,omitempty"`
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		for i := range n {
			if buf[i] == '\n' {
				count++
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return count
		}
	}
	return count
}

func parseSessionTimestamp(id core.SessionID) time.Time {
	s := string(id)
	if !strings.HasPrefix(s, "sess_") {
		return time.Time{}
	}

	s = strings.TrimPrefix(s, "sess_")
	parts := strings.SplitN(s, "_", 2)
	if len(parts) == 0 {
		return time.Time{}
	}

	t, err := time.Parse("20060102T150405.000000000", parts[0])
	if err != nil {
		return time.Time{}
	}
	return t
}
