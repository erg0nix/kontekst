package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "myskill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "myskill"
description = "Test skill"
allowed_tools = ["read_file", "list_files"]
+++

Do something with: $ARGUMENTS
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := loadSkillFile(skillPath)
	if err != nil {
		t.Fatalf("loadSkillFile failed: %v", err)
	}

	if skill.Name != "myskill" {
		t.Errorf("expected name 'myskill', got %q", skill.Name)
	}
	if skill.Description != "Test skill" {
		t.Errorf("expected description 'Test skill', got %q", skill.Description)
	}
	if len(skill.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(skill.AllowedTools))
	}
	if skill.Content != "Do something with: $ARGUMENTS" {
		t.Errorf("unexpected content: %q", skill.Content)
	}
}

func TestSkillRender(t *testing.T) {
	skill := &Skill{
		Name:    "test",
		Content: "File: $0\nAll args: $ARGUMENTS",
	}

	rendered, shellCmds, err := skill.Render("file.go extra args")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(shellCmds) != 0 {
		t.Errorf("expected no shell commands, got %d", len(shellCmds))
	}

	expected := "File: file.go\nAll args: file.go extra args"
	if rendered != expected {
		t.Errorf("expected %q, got %q", expected, rendered)
	}
}

func TestSkillRenderShellCommands(t *testing.T) {
	skill := &Skill{
		Name:    "test",
		Content: "Status: !`git status`\nBranch: !`git branch`",
	}

	_, shellCmds, err := skill.Render("")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(shellCmds) != 2 {
		t.Fatalf("expected 2 shell commands, got %d", len(shellCmds))
	}

	if shellCmds[0].Command != "git status" {
		t.Errorf("expected 'git status', got %q", shellCmds[0].Command)
	}
	if shellCmds[1].Command != "git branch" {
		t.Errorf("expected 'git branch', got %q", shellCmds[1].Command)
	}
}

func TestRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "myskill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `+++
name = "myskill"
description = "Test skill"
+++

Test content
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	skill, found := registry.Get("myskill")
	if !found {
		t.Fatal("skill not found")
	}
	if skill.Name != "myskill" {
		t.Errorf("expected name 'myskill', got %q", skill.Name)
	}

	skills := registry.ModelInvocableSkills()
	if len(skills) != 1 {
		t.Errorf("expected 1 model-invocable skill, got %d", len(skills))
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		input       string
		wantFM      string
		wantBody    string
	}{
		{
			input:    "no frontmatter",
			wantFM:   "",
			wantBody: "no frontmatter",
		},
		{
			input:    "+++\nkey = \"value\"\n+++\nbody here",
			wantFM:   "key = \"value\"",
			wantBody: "\nbody here",
		},
		{
			input:    "+++\nname = \"test\"\ndescription = \"desc\"\n+++\n\nBody content",
			wantFM:   "name = \"test\"\ndescription = \"desc\"",
			wantBody: "\n\nBody content",
		},
	}

	for _, tt := range tests {
		fm, body := parseFrontmatter(tt.input)
		if fm != tt.wantFM {
			t.Errorf("parseFrontmatter(%q): frontmatter = %q, want %q", tt.input, fm, tt.wantFM)
		}
		if body != tt.wantBody {
			t.Errorf("parseFrontmatter(%q): body = %q, want %q", tt.input, body, tt.wantBody)
		}
	}
}
