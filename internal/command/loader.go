package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type manifestArgument struct {
	Name        string `toml:"name"`
	Type        string `toml:"type"`
	Description string `toml:"description"`
	Required    bool   `toml:"required"`
	Default     string `toml:"default"`
}

type manifestPython struct {
	Requirements string `toml:"requirements"`
}

type manifest struct {
	Name        string             `toml:"name"`
	Description string             `toml:"description"`
	Runtime     string             `toml:"runtime"`
	WorkingDir  string             `toml:"working_dir"`
	Timeout     int                `toml:"timeout"`
	Arguments   []manifestArgument `toml:"arguments"`
	Python      manifestPython     `toml:"python"`
}

func loadCommandDir(dirPath string) (*Command, error) {
	manifestPath := filepath.Join(dirPath, "command.toml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	if m.WorkingDir == "" {
		m.WorkingDir = "command"
	}
	if m.Timeout <= 0 {
		m.Timeout = 60
	}
	if m.Timeout > 300 {
		m.Timeout = 300
	}

	var scriptName string
	switch m.Runtime {
	case "bash":
		scriptName = "run.sh"
	case "python":
		scriptName = "run.py"
	default:
		return nil, fmt.Errorf("invalid runtime %q: must be \"bash\" or \"python\"", m.Runtime)
	}

	scriptPath := filepath.Join(dirPath, scriptName)
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script %s not found in %s", scriptName, dirPath)
	}

	var arguments []Argument
	for _, a := range m.Arguments {
		arguments = append(arguments, Argument{
			Name:        a.Name,
			Type:        a.Type,
			Description: a.Description,
			Required:    a.Required,
			Default:     a.Default,
		})
	}

	cmd := &Command{
		Name:             m.Name,
		Description:      m.Description,
		Runtime:          m.Runtime,
		WorkingDir:       m.WorkingDir,
		Timeout:          m.Timeout,
		Arguments:        arguments,
		Dir:              dirPath,
		ScriptPath:       scriptPath,
		RequirementsFile: m.Python.Requirements,
	}

	if err := cmd.validate(); err != nil {
		return nil, fmt.Errorf("invalid command %q: %w", dirPath, err)
	}

	return cmd, nil
}
