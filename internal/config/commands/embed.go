package commands

import (
	_ "embed"
)

//go:embed content/grep-command.toml
var GrepCommandTOML string

//go:embed content/grep-run.sh
var GrepRunScript string
