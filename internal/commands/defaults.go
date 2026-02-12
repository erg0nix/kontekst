package commands

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed content/*
var bundledContent embed.FS

func EnsureDefaults(commandsDir string) error {
	entries, err := bundledContent.ReadDir("content")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cmdDir := filepath.Join(commandsDir, entry.Name())
		if _, err := os.Stat(cmdDir); err == nil {
			continue
		}

		if err := os.MkdirAll(cmdDir, 0o755); err != nil {
			return err
		}

		embeddedDir := filepath.Join("content", entry.Name())
		files, err := bundledContent.ReadDir(embeddedDir)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			data, err := bundledContent.ReadFile(filepath.Join(embeddedDir, f.Name()))
			if err != nil {
				return err
			}

			perm := fs.FileMode(0o644)
			if isExecutable(f.Name()) {
				perm = 0o755
			}

			if err := os.WriteFile(filepath.Join(cmdDir, f.Name()), data, perm); err != nil {
				return err
			}
		}
	}

	return nil
}

func isExecutable(name string) bool {
	return strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".py")
}
