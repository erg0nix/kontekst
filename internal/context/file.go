package context

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

// SessionFile provides append-only storage and tail-based loading of messages in a JSONL file.
type SessionFile struct {
	path string
	mu   sync.Mutex
}

// NewSessionFile creates a SessionFile that reads from and writes to the given path.
func NewSessionFile(path string) *SessionFile {
	return &SessionFile{path: path}
}

// Append writes a single message as a JSON line to the end of the session file.
func (sf *SessionFile) Append(msg core.Message) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	file, err := os.OpenFile(sf.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(msg)
}

// LoadTail reads messages from the end of the file until the token budget is exhausted.
func (sf *SessionFile) LoadTail(tokenBudget int) ([]core.Message, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	file, err := os.Open(sf.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil, nil
	}

	var messages []core.Message
	tokensUsed := 0
	remaining := fileSize
	var carryover []byte

	for remaining > 0 {
		readSize := int64(chunkSize)
		if readSize > remaining {
			readSize = remaining
		}

		offset := remaining - readSize
		chunk := make([]byte, readSize)

		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}

		if _, err := io.ReadFull(file, chunk); err != nil {
			return nil, err
		}

		if len(carryover) > 0 {
			chunk = append(chunk, carryover...)
			carryover = nil
		}

		lines := splitLines(chunk)

		if offset > 0 && len(lines) > 0 {
			carryover = lines[0]
			lines = lines[1:]
		}

		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			if len(line) == 0 {
				continue
			}

			var msg core.Message
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}

			if tokensUsed+msg.Tokens > tokenBudget && len(messages) > 0 {
				slices.Reverse(messages)
				return messages, nil
			}

			messages = append(messages, msg)
			tokensUsed += msg.Tokens
		}

		remaining = offset
	}

	if len(carryover) > 0 {
		var msg core.Message
		if err := json.Unmarshal(carryover, &msg); err == nil {
			if tokensUsed+msg.Tokens <= tokenBudget || len(messages) == 0 {
				messages = append(messages, msg)
			}
		}
	}

	slices.Reverse(messages)

	return messages, nil
}

const chunkSize = 8 * 1024

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}

	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}
