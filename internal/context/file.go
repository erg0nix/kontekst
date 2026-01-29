package context

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

type FileContextService struct {
	BaseDir        string
	SystemTemplate string
	UserTemplate   string
	MaxTokens      int
}

func (service *FileContextService) NewWindow(sessionID core.SessionID) (ContextWindow, error) {
	sessionDir := filepath.Join(service.BaseDir, "sessions")

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(sessionDir, string(sessionID)+".jsonl")

	return NewFileContext(sessionPath, service.SystemTemplate, service.UserTemplate, service.MaxTokens)
}

type FileContext struct {
	Path           string
	mu             sync.Mutex
	Messages       []core.Message
	SystemTemplate string
	UserTemplate   string
	MaxTokens      int
}

func NewFileContext(path string, systemTemplate string, userTemplate string, maxTokens int) (*FileContext, error) {
	ctx := &FileContext{
		Path:           path,
		SystemTemplate: systemTemplate,
		UserTemplate:   userTemplate,
		MaxTokens:      maxTokens,
	}

	if err := ctx.load(); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (fileContext *FileContext) AddMessage(msg core.Message) error {
	fileContext.mu.Lock()
	defer fileContext.mu.Unlock()

	fileContext.Messages = append(fileContext.Messages, msg)

	return fileContext.append(msg)
}

func (fileContext *FileContext) BuildContext(_ func(string) (int, error)) ([]core.Message, error) {
	fileContext.mu.Lock()
	defer fileContext.mu.Unlock()

	systemMessage := core.Message{Role: core.RoleSystem, Content: fileContext.SystemTemplate}
	out := []core.Message{systemMessage}
	out = append(out, fileContext.Messages...)

	return out, nil
}

func (fileContext *FileContext) RenderUserMessage(prompt string) (string, error) {
	if fileContext.UserTemplate == "" {
		return prompt, nil
	}

	return strings.ReplaceAll(fileContext.UserTemplate, "{{ user_message }}", prompt), nil
}

func (fileContext *FileContext) AddToolResult(result core.ToolResult) error {
	msg := core.Message{Role: core.RoleTool, Content: result.Output, ToolResult: &result}

	return fileContext.AddMessage(msg)
}

func (fileContext *FileContext) load() error {
	file, err := os.Open(fileContext.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

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

		fileContext.Messages = append(fileContext.Messages, msg)
	}

	return scanner.Err()
}

func (fileContext *FileContext) append(msg core.Message) error {
	file, err := os.OpenFile(fileContext.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	return encoder.Encode(msg)
}
