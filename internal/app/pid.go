// Package app provides server lifecycle management and service wiring.
package app

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// ReadPID reads a PID from the given file and returns it if the process is alive, or 0 otherwise.
func ReadPID(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return 0
	}

	if process.Signal(syscall.Signal(0)) != nil {
		return 0
	}

	return pid
}

// FindProcessPID searches for a running process by name and returns its PID, or 0 if not found.
func FindProcessPID(name string) int {
	out, err := pidofCommand(name)
	if err != nil {
		return 0
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0
	}

	pid, _ := strconv.Atoi(fields[0])
	return pid
}

func pidofCommand(name string) ([]byte, error) {
	return exec.Command("pgrep", "-f", name).Output()
}
