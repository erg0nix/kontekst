package conversation

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/erg0nix/kontekst/internal/core"
)

// BudgetParams holds the token budget breakdown used to size the context window for a run.
type BudgetParams struct {
	ContextSize      int
	SystemContent    string
	SystemTokens     int
	ToolTokens       int
	UserPromptTokens int
}

// Window manages the message history and system prompt for a single session's conversation context.
type Window interface {
	SystemContent() string
	StartRun(params BudgetParams) error
	CompleteRun()
	AddMessage(msg core.Message) error
	BuildContext() ([]core.Message, error)
	SetAgentSystemPrompt(prompt string)
	SetActiveSkill(skill *core.SkillMetadata)
	ActiveSkill() *core.SkillMetadata
	Snapshot() core.ContextSnapshot
}

// Service creates Window instances for session.
type Service interface {
	NewWindow(sessionID core.SessionID) (Window, error)
}

// FileService implements Service using JSONL session files on disk.
type FileService struct {
	dataDir string
}

// NewFileService creates a FileService rooted at the given data directory.
func NewFileService(dataDir string) *FileService {
	return &FileService{dataDir: dataDir}
}

// NewWindow creates a new Window backed by the session's JSONL file.
func (service *FileService) NewWindow(sessionID core.SessionID) (Window, error) {
	sessionDir := filepath.Join(service.dataDir, "sessions")

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(sessionDir, string(sessionID)+".jsonl")
	sessionFile := NewSessionFile(sessionPath)

	return newContextWindow(sessionFile), nil
}

type contextWindow struct {
	sessionFile       *SessionFile
	history           []core.Message
	memory            []core.Message
	agentSystemPrompt string
	activeSkill       *core.SkillMetadata
	systemContent     string
	contextSize       int
	systemTokens      int
	toolTokens        int
	mu                sync.Mutex
}

func newContextWindow(sessionFile *SessionFile) *contextWindow {
	return &contextWindow{
		sessionFile: sessionFile,
	}
}

func (cw *contextWindow) SystemContent() string {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	content := cw.agentSystemPrompt

	if cw.activeSkill != nil {
		content = content + fmt.Sprintf("\n\n<active-skill name=%q path=%q />", cw.activeSkill.Name, cw.activeSkill.Path)
	}

	return content
}

func (cw *contextWindow) StartRun(params BudgetParams) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.contextSize = params.ContextSize
	cw.systemContent = params.SystemContent
	cw.systemTokens = params.SystemTokens
	cw.toolTokens = params.ToolTokens

	historyBudget := cw.contextSize - params.SystemTokens - params.ToolTokens - params.UserPromptTokens
	if historyBudget < 0 {
		historyBudget = 0
	}

	history, err := cw.sessionFile.LoadTail(historyBudget)
	if err != nil {
		return err
	}

	cw.history = history
	cw.memory = nil

	return nil
}

func (cw *contextWindow) CompleteRun() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.memory = nil
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
	totalTokens := cw.systemTokens + cw.toolTokens + historyTokens + memoryTokens

	historyBudget := cw.contextSize - cw.systemTokens - cw.toolTokens - memoryTokens
	if historyBudget < 0 {
		historyBudget = 0
	}

	messages := make([]core.MessageStats, 0, 1+len(cw.history)+len(cw.memory))
	messages = append(messages, core.MessageStats{Role: core.RoleSystem, Tokens: cw.systemTokens, Source: "system"})
	for _, msg := range cw.history {
		messages = append(messages, core.MessageStats{Role: msg.Role, Tokens: msg.Tokens, Source: "history"})
	}
	for _, msg := range cw.memory {
		messages = append(messages, core.MessageStats{Role: msg.Role, Tokens: msg.Tokens, Source: "memory"})
	}

	return core.ContextSnapshot{
		ContextSize:     cw.contextSize,
		SystemTokens:    cw.systemTokens,
		ToolTokens:      cw.toolTokens,
		HistoryTokens:   historyTokens,
		MemoryTokens:    memoryTokens,
		TotalTokens:     totalTokens,
		RemainingTokens: cw.contextSize - totalTokens,
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
