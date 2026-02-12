package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultsCreatesGrepCommand(t *testing.T) {
	commandsDir := t.TempDir()

	if err := EnsureDefaults(commandsDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	cmdDir := filepath.Join(commandsDir, "grep")

	tomlPath := filepath.Join(cmdDir, "command.toml")
	if _, err := os.Stat(tomlPath); err != nil {
		t.Fatalf("command.toml not found: %v", err)
	}

	scriptPath := filepath.Join(cmdDir, "run.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("run.sh not found: %v", err)
	}

	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("run.sh is not executable: %v", info.Mode().Perm())
	}
}

func TestEnsureDefaultsIdempotent(t *testing.T) {
	commandsDir := t.TempDir()

	if err := EnsureDefaults(commandsDir); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if err := EnsureDefaults(commandsDir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestEnsureDefaultsSkipsExisting(t *testing.T) {
	commandsDir := t.TempDir()

	cmdDir := filepath.Join(commandsDir, "grep")
	os.MkdirAll(cmdDir, 0o755)
	customToml := []byte("name = \"my-grep\"\n")
	os.WriteFile(filepath.Join(cmdDir, "command.toml"), customToml, 0o644)

	if err := EnsureDefaults(commandsDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cmdDir, "command.toml"))
	if err != nil {
		t.Fatalf("reading command.toml: %v", err)
	}
	if string(data) != string(customToml) {
		t.Error("EnsureDefaults overwrote existing command")
	}
}

func TestEnsureDefaultsLoadableByRegistry(t *testing.T) {
	commandsDir := t.TempDir()

	if err := EnsureDefaults(commandsDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	registry := NewRegistry(commandsDir)
	if err := registry.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cmd, ok := registry.Get("grep")
	if !ok {
		t.Fatal("grep command not found in registry")
	}

	if cmd.Name != "grep" {
		t.Errorf("unexpected name: %q", cmd.Name)
	}
	if cmd.Runtime != "bash" {
		t.Errorf("unexpected runtime: %q", cmd.Runtime)
	}
	if cmd.WorkingDir != "agent" {
		t.Errorf("unexpected working_dir: %q", cmd.WorkingDir)
	}

	hasPattern := false
	for _, arg := range cmd.Arguments {
		if arg.Name == "pattern" && arg.Required {
			hasPattern = true
		}
	}
	if !hasPattern {
		t.Error("grep command missing required 'pattern' argument")
	}
}
