// Package command provides default slash command configuration and installation.
package command

import (
	"os"
	"path/filepath"
)

type bundledCommand struct {
	name       string
	config     string
	script     string
	scriptName string
}

var bundledCommands = []bundledCommand{
	{
		name:       "grep",
		config:     GrepCommandTOML,
		script:     GrepRunScript,
		scriptName: "run.sh",
	},
}

// EnsureDefaults creates bundled command configurations under commandsDir if they do not already exist.
func EnsureDefaults(commandsDir string) error {
	for _, c := range bundledCommands {
		if err := ensureCommand(commandsDir, c); err != nil {
			return err
		}
	}
	return nil
}

func ensureCommand(commandsDir string, c bundledCommand) error {
	cmdDir := filepath.Join(commandsDir, c.name)

	if _, err := os.Stat(cmdDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(cmdDir, "command.toml"), []byte(c.config), 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(cmdDir, c.scriptName), []byte(c.script), 0o755)
}
