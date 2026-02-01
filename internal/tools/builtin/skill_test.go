package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/skills"
)

func TestSkillToolDescription(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "myskill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "myskill"
description = "Does something useful"
+++

Content here
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := skills.NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatal(err)
	}

	tool := &SkillTool{Registry: registry}

	desc := tool.Description()
	if !strings.Contains(desc, "myskill") {
		t.Errorf("description should contain skill name, got: %s", desc)
	}
	if !strings.Contains(desc, "Does something useful") {
		t.Errorf("description should contain skill description, got: %s", desc)
	}
	if !strings.Contains(desc, "<available_skills>") {
		t.Errorf("description should contain available_skills tag, got: %s", desc)
	}
}

func TestSkillToolExecute(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "echo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "echo"
description = "Echoes arguments"
+++

Echo: $ARGUMENTS
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := skills.NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatal(err)
	}

	tool := &SkillTool{Registry: registry}

	var injectedMsg core.Message

	callbacks := &SkillCallbacks{
		ContextInjector: func(msg core.Message) error {
			injectedMsg = msg
			return nil
		},
	}

	ctx := WithSkillCallbacks(context.Background(), callbacks)

	result, err := tool.Execute(map[string]any{
		"name":      "echo",
		"arguments": "hello world",
	}, ctx)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result, "echo") {
		t.Errorf("result should mention skill name, got: %s", result)
	}

	if injectedMsg.Role != core.RoleUser {
		t.Errorf("injected message should be user role, got: %s", injectedMsg.Role)
	}

	if !strings.Contains(injectedMsg.Content, "Echo: hello world") {
		t.Errorf("injected content should contain rendered skill, got: %s", injectedMsg.Content)
	}
}

func TestSkillToolExecuteNotFound(t *testing.T) {
	registry := skills.NewRegistry(t.TempDir())
	_ = registry.Load()

	tool := &SkillTool{Registry: registry}

	callbacks := &SkillCallbacks{
		ContextInjector: func(msg core.Message) error { return nil },
	}
	ctx := WithSkillCallbacks(context.Background(), callbacks)

	_, err := tool.Execute(map[string]any{"name": "nonexistent"}, ctx)
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestSkillToolIntegrationWithActiveSkill(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "review")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "review"
description = "Review pull request"
+++

Review PR #$0

Steps:
1. Check the diff
2. Look for issues
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := skills.NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatal(err)
	}

	tool := &SkillTool{Registry: registry}

	var injectedMsg core.Message
	var activeSkill *core.SkillMetadata

	callbacks := &SkillCallbacks{
		ContextInjector: func(msg core.Message) error {
			injectedMsg = msg
			return nil
		},
		SetActiveSkill: func(skill *core.SkillMetadata) {
			activeSkill = skill
		},
	}

	ctx := WithSkillCallbacks(context.Background(), callbacks)

	result, err := tool.Execute(map[string]any{
		"name":      "review",
		"arguments": "123",
	}, ctx)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result, "review") {
		t.Errorf("result should mention skill name, got: %s", result)
	}

	if activeSkill == nil {
		t.Fatal("SetActiveSkill callback was not called")
	}
	if activeSkill.Name != "review" {
		t.Errorf("active skill name should be 'review', got: %s", activeSkill.Name)
	}
	if activeSkill.Path != skillDir {
		t.Errorf("active skill path mismatch, want: %s, got: %s", skillDir, activeSkill.Path)
	}

	if !strings.Contains(injectedMsg.Content, "[Skill: review]") {
		t.Errorf("injected content should contain skill header, got: %s", injectedMsg.Content)
	}
	if !strings.Contains(injectedMsg.Content, "Base path:") {
		t.Errorf("injected content should contain base path, got: %s", injectedMsg.Content)
	}
	if !strings.Contains(injectedMsg.Content, "Review PR #123") {
		t.Errorf("injected content should contain rendered argument, got: %s", injectedMsg.Content)
	}
}

func TestSkillToolExecuteDisabledModelInvocation(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "useronly")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "useronly"
description = "User only skill"
disable_model_invocation = true
+++

Content
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := skills.NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatal(err)
	}

	tool := &SkillTool{Registry: registry}

	callbacks := &SkillCallbacks{
		ContextInjector: func(msg core.Message) error { return nil },
	}
	ctx := WithSkillCallbacks(context.Background(), callbacks)

	_, err := tool.Execute(map[string]any{"name": "useronly"}, ctx)
	if err == nil {
		t.Error("expected error for disabled model invocation skill")
	}
	if !strings.Contains(err.Error(), "can only be invoked by user") {
		t.Errorf("error should mention user-only invocation, got: %v", err)
	}
}
