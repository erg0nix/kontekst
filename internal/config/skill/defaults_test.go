package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultsCreatesKontekstSkill(t *testing.T) {
	skillsDir := t.TempDir()

	if err := EnsureDefaults(skillsDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	skillPath := filepath.Join(skillsDir, "kontekst", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("kontekst SKILL.md not found: %v", err)
	}

	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("SKILL.md is empty")
	}
}

func TestEnsureDefaultsIdempotent(t *testing.T) {
	skillsDir := t.TempDir()

	if err := EnsureDefaults(skillsDir); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if err := EnsureDefaults(skillsDir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestEnsureDefaultsSkipsExisting(t *testing.T) {
	skillsDir := t.TempDir()

	skillDir := filepath.Join(skillsDir, "kontekst")
	os.MkdirAll(skillDir, 0o755)
	customContent := []byte("my custom skill\n")
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), customContent, 0o644)

	if err := EnsureDefaults(skillsDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	if string(data) != string(customContent) {
		t.Error("EnsureDefaults overwrote existing skill")
	}
}
