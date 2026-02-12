package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctx "github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/skills"
)

type capturingContext struct {
	mockContext
	capturedPrompt string
}

func (c *capturingContext) AddMessage(msg core.Message) error {
	if msg.Role == core.RoleUser && c.capturedPrompt == "" {
		c.capturedPrompt = msg.Content
	}
	return c.mockContext.AddMessage(msg)
}

type mockContextService struct {
	window ctx.ContextWindow
}

func (m *mockContextService) NewWindow(_ core.SessionID) (ctx.ContextWindow, error) {
	return m.window, nil
}

type mockSessionService struct{}

func (m *mockSessionService) Create() (core.SessionID, string, error) {
	return "test-session", "/tmp/test", nil
}

func (m *mockSessionService) Ensure(_ core.SessionID) (string, error) {
	return "/tmp/test", nil
}

func (m *mockSessionService) GetDefaultAgent(_ core.SessionID) (string, error) {
	return "", nil
}

func (m *mockSessionService) SetDefaultAgent(_ core.SessionID, _ string) error {
	return nil
}

func drainEvents(events <-chan AgentEvent) {
	for range events {
	}
}

func TestStartRun_WithAgentsMD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Always use tabs."), 0o644); err != nil {
		t.Fatal(err)
	}

	capturer := &capturingContext{}
	runner := &AgentRunner{
		Tools:    &mockToolExecutor{},
		Context:  &mockContextService{window: capturer},
		Sessions: &mockSessionService{},
	}

	_, events, err := runner.StartRun(RunConfig{
		Prompt:     "hello",
		WorkingDir: dir,
	})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	drainEvents(events)

	if !strings.Contains(capturer.capturedPrompt, "<project-instructions>") {
		t.Fatal("expected <project-instructions> tag in prompt")
	}
	if !strings.Contains(capturer.capturedPrompt, "Always use tabs.") {
		t.Fatal("expected AGENTS.md content in prompt")
	}
	if !strings.Contains(capturer.capturedPrompt, "hello") {
		t.Fatal("expected original prompt to be preserved")
	}
}

func TestStartRun_WithoutAgentsMD(t *testing.T) {
	dir := t.TempDir()

	capturer := &capturingContext{}
	runner := &AgentRunner{
		Tools:    &mockToolExecutor{},
		Context:  &mockContextService{window: capturer},
		Sessions: &mockSessionService{},
	}

	_, events, err := runner.StartRun(RunConfig{
		Prompt:     "hello",
		WorkingDir: dir,
	})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	drainEvents(events)

	if strings.Contains(capturer.capturedPrompt, "<project-instructions>") {
		t.Fatal("did not expect <project-instructions> tag when no AGENTS.md exists")
	}
	if capturer.capturedPrompt != "hello" {
		t.Fatalf("expected unmodified prompt 'hello', got %q", capturer.capturedPrompt)
	}
}

func TestStartRun_AgentsMDBeforeSkillContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Project rules here."), 0o644); err != nil {
		t.Fatal(err)
	}

	capturer := &capturingContext{}
	runner := &AgentRunner{
		Tools:    &mockToolExecutor{},
		Context:  &mockContextService{window: capturer},
		Sessions: &mockSessionService{},
	}

	_, events, err := runner.StartRun(RunConfig{
		Prompt:       "do something",
		WorkingDir:   dir,
		Skill:        &skills.Skill{Name: "test-skill", Path: "/test"},
		SkillContent: "Skill instructions here.",
	})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	drainEvents(events)

	projectIdx := strings.Index(capturer.capturedPrompt, "<project-instructions>")
	skillIdx := strings.Index(capturer.capturedPrompt, "[Skill: test-skill]")
	promptIdx := strings.Index(capturer.capturedPrompt, "do something")

	if projectIdx == -1 {
		t.Fatal("expected <project-instructions> in prompt")
	}
	if skillIdx == -1 {
		t.Fatal("expected skill content in prompt")
	}
	if projectIdx >= skillIdx {
		t.Fatal("expected project instructions to appear before skill content")
	}
	if skillIdx >= promptIdx {
		t.Fatal("expected skill content to appear before user prompt")
	}
}
