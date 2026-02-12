package skills

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed content/*
var bundledContent embed.FS

func EnsureDefaults(skillsDir string) error {
	entries, err := bundledContent.ReadDir("content")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsDir, entry.Name())
		if _, err := os.Stat(skillDir); err == nil {
			continue
		}

		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return err
		}

		data, err := bundledContent.ReadFile(filepath.Join("content", entry.Name(), "SKILL.md"))
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), data, 0o644); err != nil {
			return err
		}
	}

	return nil
}
