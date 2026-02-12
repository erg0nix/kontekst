package main

import "os/exec"

func pidofCommand(name string) ([]byte, error) {
	return exec.Command("pgrep", "-f", name).Output()
}
