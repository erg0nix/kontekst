package skill

import (
	"os"
	"path/filepath"
)

type bundledSkill struct {
	name    string
	content string
}

var bundledSkills = []bundledSkill{
	{name: "kontekst", content: KontekstSkillContent},
}

// EnsureDefaults creates bundled skill files under skillsDir if they do not already exist.
func EnsureDefaults(skillsDir string) error {
	for _, s := range bundledSkills {
		if err := ensureSkill(skillsDir, s); err != nil {
			return err
		}
	}
	return nil
}

func ensureSkill(skillsDir string, s bundledSkill) error {
	skillDir := filepath.Join(skillsDir, s.name)

	if _, err := os.Stat(skillDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(s.content), 0o644)
}
