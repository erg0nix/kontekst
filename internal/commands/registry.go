package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Registry loads and stores commands from a directory, providing thread-safe access by name.
type Registry struct {
	commandsDir string
	commands    map[string]*Command
	mu          sync.RWMutex
}

// NewRegistry creates a Registry that loads commands from the given directory.
func NewRegistry(commandsDir string) *Registry {
	return &Registry{
		commandsDir: commandsDir,
		commands:    make(map[string]*Command),
	}
}

// Load discovers and parses all command directories from the registry's base directory.
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.commands = make(map[string]*Command)

	if _, err := os.Stat(r.commandsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(r.commandsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(r.commandsDir, entry.Name())
		cmd, err := loadCommandDir(dirPath)
		if err != nil {
			slog.Warn("skipping command", "dir", entry.Name(), "error", err)
			continue
		}
		r.commands[cmd.Name] = cmd
	}

	return nil
}

// Get returns the command with the given name, or false if not found.
func (r *Registry) Get(name string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, ok := r.commands[name]
	return cmd, ok
}

// Summaries returns a formatted string listing all commands with their descriptions and arguments.
func (r *Registry) Summaries() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.commands) == 0 {
		return ""
	}

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		cmd := r.commands[name]
		sb.WriteString(fmt.Sprintf("- %s: %s\n", cmd.Name, cmd.Description))
		if len(cmd.Arguments) > 0 {
			sb.WriteString("  Arguments: ")
			for i, arg := range cmd.Arguments {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s (%s", arg.Name, arg.Type))
				if arg.Required {
					sb.WriteString(", required")
				} else {
					sb.WriteString(", optional")
				}
				if arg.Default != "" {
					sb.WriteString(fmt.Sprintf(", default: %q", arg.Default))
				}
				sb.WriteString(")")
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
