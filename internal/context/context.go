package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/core"
)

type ContextWindow interface {
	SystemContent() string
	StartRun(systemContent string, systemTokens int) error
	CompleteRun() error
	AddMessage(msg core.Message) error
	BuildContext() ([]core.Message, error)
	RenderUserMessage(prompt string) (string, error)
	SetAgentSystemPrompt(prompt string)
	SetActiveSkill(skill *core.SkillMetadata)
	ActiveSkill() *core.SkillMetadata
	Snapshot() core.ContextSnapshot
}

type ContextService interface {
	NewWindow(sessionID core.SessionID) (ContextWindow, error)
}

type FileContextService struct {
	cfg *config.Config
}

func NewFileContextService(cfg *config.Config) *FileContextService {
	return &FileContextService{cfg: cfg}
}

func (service *FileContextService) NewWindow(sessionID core.SessionID) (ContextWindow, error) {
	sessionDir := filepath.Join(service.cfg.DataDir, "sessions")

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(sessionDir, string(sessionID)+".jsonl")
	sessionFile := NewSessionFile(sessionPath)

	return newContextWindow(sessionFile, service.cfg), nil
}

type contextWindow struct {
	cfg               *config.Config
	sessionFile       *SessionFile
	history           []core.Message
	memory            []core.Message
	agentSystemPrompt string
	activeSkill       *core.SkillMetadata
	systemContent     string
	systemTokens      int
	userTemplate      string
	mu                sync.Mutex
}

func newContextWindow(sessionFile *SessionFile, cfg *config.Config) *contextWindow {
	return &contextWindow{
		cfg:          cfg,
		sessionFile:  sessionFile,
		userTemplate: defaultUserTemplate,
	}
}

const (
	defaultUserTemplate = "{{ user_message }}"
	historyRatio        = 0.30
)

func (cw *contextWindow) SystemContent() string {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	content := cw.agentSystemPrompt

	if cw.activeSkill != nil {
		content = content + fmt.Sprintf("\n\n<active-skill name=%q path=%q />", cw.activeSkill.Name, cw.activeSkill.Path)
	}

	return content
}

func (cw *contextWindow) StartRun(systemContent string, systemTokens int) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.systemContent = systemContent
	cw.systemTokens = systemTokens

	remaining := cw.cfg.ContextSize - systemTokens
	historyBudget := int(float64(remaining) * historyRatio)

	history, err := cw.sessionFile.LoadTail(historyBudget)
	if err != nil {
		return err
	}

	cw.history = history
	cw.memory = nil

	return nil
}

func (cw *contextWindow) CompleteRun() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.memory = nil

	return nil
}

func (cw *contextWindow) AddMessage(msg core.Message) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if err := cw.sessionFile.Append(msg); err != nil {
		return err
	}

	cw.memory = append(cw.memory, msg)

	return nil
}

func (cw *contextWindow) BuildContext() ([]core.Message, error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	systemMessage := core.Message{Role: core.RoleSystem, Content: cw.systemContent}

	out := []core.Message{systemMessage}
	out = append(out, cw.history...)
	out = append(out, cw.memory...)

	return out, nil
}

func (cw *contextWindow) RenderUserMessage(prompt string) (string, error) {
	if cw.userTemplate == "" {
		return prompt, nil
	}

	return strings.ReplaceAll(cw.userTemplate, "{{ user_message }}", prompt), nil
}

func (cw *contextWindow) SetAgentSystemPrompt(prompt string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.agentSystemPrompt = prompt
}

func (cw *contextWindow) SetActiveSkill(skill *core.SkillMetadata) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.activeSkill = skill
}

func (cw *contextWindow) ActiveSkill() *core.SkillMetadata {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	return cw.activeSkill
}

func (cw *contextWindow) Snapshot() core.ContextSnapshot {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	historyTokens := sumTokens(cw.history)
	memoryTokens := sumTokens(cw.memory)
	totalTokens := cw.systemTokens + historyTokens + memoryTokens

	remaining := cw.cfg.ContextSize - cw.systemTokens
	historyBudget := int(float64(remaining) * historyRatio)

	messages := make([]core.MessageStats, 0, 1+len(cw.history)+len(cw.memory))
	messages = append(messages, core.MessageStats{Role: core.RoleSystem, Tokens: cw.systemTokens, Source: "system"})
	for _, msg := range cw.history {
		messages = append(messages, core.MessageStats{Role: msg.Role, Tokens: msg.Tokens, Source: "history"})
	}
	for _, msg := range cw.memory {
		messages = append(messages, core.MessageStats{Role: msg.Role, Tokens: msg.Tokens, Source: "memory"})
	}

	return core.ContextSnapshot{
		ContextSize:     cw.cfg.ContextSize,
		SystemTokens:    cw.systemTokens,
		HistoryTokens:   historyTokens,
		MemoryTokens:    memoryTokens,
		TotalTokens:     totalTokens,
		RemainingTokens: cw.cfg.ContextSize - totalTokens,
		HistoryMessages: len(cw.history),
		MemoryMessages:  len(cw.memory),
		TotalMessages:   1 + len(cw.history) + len(cw.memory),
		HistoryBudget:   historyBudget,
		Messages:        messages,
	}
}

func sumTokens(messages []core.Message) int {
	total := 0
	for _, msg := range messages {
		total += msg.Tokens
	}
	return total
}
