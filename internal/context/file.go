package context

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

type SessionFile struct {
	path string
	mu   sync.Mutex
}

func NewSessionFile(path string) *SessionFile {
	return &SessionFile{path: path}
}

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

	lines, err := readLinesBackward(file, fileSize)
	if err != nil {
		return nil, err
	}

	var messages []core.Message
	tokensUsed := 0

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
			break
		}

		messages = append([]core.Message{msg}, messages...)
		tokensUsed += msg.Tokens
	}

	return messages, nil
}

func (sf *SessionFile) LoadAll() ([]core.Message, error) {
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

	var messages []core.Message
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg core.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}

const chunkSize = 8 * 1024

func readLinesBackward(file *os.File, fileSize int64) ([][]byte, error) {
	var allData []byte
	remaining := fileSize

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

		allData = append(chunk, allData...)
		remaining = offset
	}

	var lines [][]byte
	start := 0

	for i := 0; i < len(allData); i++ {
		if allData[i] == '\n' {
			if i > start {
				lines = append(lines, allData[start:i])
			}
			start = i + 1
		}
	}

	if start < len(allData) {
		lines = append(lines, allData[start:])
	}

	return lines, nil
}
