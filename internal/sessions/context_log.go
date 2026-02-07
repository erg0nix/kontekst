package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

type ContextLogEntry struct {
	Timestamp time.Time            `json:"ts"`
	RunID     core.RunID           `json:"run_id"`
	Turn      int                  `json:"turn"`
	Snapshot  core.ContextSnapshot `json:"snapshot"`
}

type ContextLogWriter struct {
	baseDir string
	mu      sync.Mutex
}

func NewContextLogWriter(baseDir string) *ContextLogWriter {
	return &ContextLogWriter{baseDir: baseDir}
}

func (w *ContextLogWriter) Write(runID core.RunID, turn int, snapshot core.ContextSnapshot) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	dir := filepath.Join(w.baseDir, "runs", string(runID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(dir, "context.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	entry := ContextLogEntry{
		Timestamp: time.Now().UTC(),
		RunID:     runID,
		Turn:      turn,
		Snapshot:  snapshot,
	}

	return json.NewEncoder(file).Encode(entry)
}
