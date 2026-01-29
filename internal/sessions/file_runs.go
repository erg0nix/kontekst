package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

type RunRecord struct {
	RunID     core.RunID     `json:"run_id"`
	SessionID core.SessionID `json:"session_id"`
	Status    string         `json:"status"`
	Timestamp time.Time      `json:"ts"`
}

type FileRunService struct {
	Path string
	mu   sync.Mutex
}

func (service *FileRunService) StartRun(sessionID core.SessionID, runID core.RunID) error {
	return service.append(RunRecord{
		RunID:     runID,
		SessionID: sessionID,
		Status:    "started",
		Timestamp: time.Now().UTC(),
	})
}

func (service *FileRunService) CompleteRun(runID core.RunID) error {
	return service.append(RunRecord{
		RunID:     runID,
		Status:    "completed",
		Timestamp: time.Now().UTC(),
	})
}

func (service *FileRunService) CancelRun(runID core.RunID) error {
	return service.append(RunRecord{
		RunID:     runID,
		Status:    "cancelled",
		Timestamp: time.Now().UTC(),
	})
}

func (service *FileRunService) FailRun(runID core.RunID) error {
	return service.append(RunRecord{
		RunID:     runID,
		Status:    "failed",
		Timestamp: time.Now().UTC(),
	})
}

func (service *FileRunService) append(record RunRecord) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	dir := filepath.Dir(service.Path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(service.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	return encoder.Encode(record)
}
