package skills

import (
	"os"
	"path/filepath"
	"strings"
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
	if skill.Content != "Do something with: $ARGUMENTS" {
		t.Errorf("unexpected content: %q", skill.Content)
	}
}

func TestSkillRender(t *testing.T) {
	skill := &Skill{
		Name:    "test",
		Content: "File: $0\nAll args: $ARGUMENTS",
	}

	rendered, err := skill.Render("file.go extra args")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	expected := "File: file.go\nAll args: file.go extra args"
	if rendered != expected {
		t.Errorf("expected %q, got %q", expected, rendered)
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

func TestRegistrySummaries(t *testing.T) {
	tmpDir := t.TempDir()

	skill1Content := `+++
name = "commit"
description = "Create a commit"
+++

Commit skill content
`
	skill2Content := `+++
name = "review"
description = "Review code changes"
+++

Review skill content
`
	skill3Content := `+++
name = "hidden"
description = "Hidden skill"
disable_model_invocation = true
+++

Hidden skill content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "commit.md"), []byte(skill1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "review.md"), []byte(skill2Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "hidden.md"), []byte(skill3Content), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	summaries := registry.Summaries()

	if summaries == "" {
		t.Fatal("Summaries returned empty string")
	}

	if !strings.Contains(summaries, "- commit: Create a commit") {
		t.Errorf("Summaries missing commit skill: %q", summaries)
	}
	if !strings.Contains(summaries, "- review: Review code changes") {
		t.Errorf("Summaries missing review skill: %q", summaries)
	}
	if strings.Contains(summaries, "hidden") {
		t.Errorf("Summaries should not include hidden skill: %q", summaries)
	}
}

func TestRegistrySummariesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	registry := NewRegistry(tmpDir)
	if err := registry.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	summaries := registry.Summaries()
	if summaries != "" {
		t.Errorf("Expected empty summaries for empty registry, got %q", summaries)
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		input    string
		wantFM   string
		wantBody string
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
