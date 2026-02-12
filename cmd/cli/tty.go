package main

import (
	"os"

	"github.com/charmbracelet/x/term"
)

func isInteractive() bool {
	return term.IsTerminal(os.Stdout.Fd())
}
