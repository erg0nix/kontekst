// Package command provides slash command loading and execution.
package command

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Argument describes a single parameter accepted by a command.
type Argument struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     string
}

// Command represents a user-defined script that can be executed by the agent.
type Command struct {
	Name             string
	Description      string
	Runtime          string
	WorkingDir       string
	Timeout          int
	Arguments        []Argument
	Dir              string
	ScriptPath       string
	RequirementsFile string
}

func (c *Command) validate() error {
	if c.Name == "" {
		return fmt.Errorf("command name is required")
	}

	if c.Runtime != "bash" && c.Runtime != "python" {
		return fmt.Errorf("invalid runtime %q: must be \"bash\" or \"python\"", c.Runtime)
	}

	if c.WorkingDir != "command" && c.WorkingDir != "agent" {
		return fmt.Errorf("invalid working_dir %q: must be \"command\" or \"agent\"", c.WorkingDir)
	}

	if c.RequirementsFile != "" {
		clean := filepath.Clean(c.RequirementsFile)
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			return fmt.Errorf("requirements file %q must be a relative path within the command directory", c.RequirementsFile)
		}
	}

	for _, arg := range c.Arguments {
		if arg.Name == "" {
			return fmt.Errorf("argument name is required")
		}
		if arg.Type != "string" {
			return fmt.Errorf("argument %q has invalid type %q: only \"string\" is supported", arg.Name, arg.Type)
		}
	}

	return nil
}
