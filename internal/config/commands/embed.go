package commands

import (
	_ "embed"
)

// GrepCommandTOML is the embedded TOML configuration for the grep command.
//
//go:embed content/grep-command.toml
var GrepCommandTOML string

// GrepRunScript is the embedded shell script that executes the grep command.
//
//go:embed content/grep-run.sh
var GrepRunScript string
