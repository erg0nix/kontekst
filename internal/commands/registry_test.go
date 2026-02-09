package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadValidBashCommand(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "echo-test")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "echo-test"
description = "Echoes a message"
runtime = "bash"

[[arguments]]
name = "message"
type = "string"
description = "Message to echo"
required = true
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), `#!/bin/bash
echo "$KONTEKST_ARG_MESSAGE"
`)

	cmd, err := loadCommandDir(cmdDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Name != "echo-test" {
		t.Errorf("name = %q, want %q", cmd.Name, "echo-test")
	}
	if cmd.Runtime != "bash" {
		t.Errorf("runtime = %q, want %q", cmd.Runtime, "bash")
	}
	if cmd.WorkingDir != "command" {
		t.Errorf("working_dir = %q, want %q", cmd.WorkingDir, "command")
	}
	if cmd.Timeout != 60 {
		t.Errorf("timeout = %d, want %d", cmd.Timeout, 60)
	}
	if len(cmd.Arguments) != 1 {
		t.Fatalf("len(arguments) = %d, want 1", len(cmd.Arguments))
	}
	if cmd.Arguments[0].Name != "message" {
		t.Errorf("argument name = %q, want %q", cmd.Arguments[0].Name, "message")
	}
	if !cmd.Arguments[0].Required {
		t.Error("argument should be required")
	}
}

func TestLoadValidPythonCommand(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "greet")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "greet"
description = "Greets by name"
runtime = "python"
timeout = 30

[python]
requirements = "requirements.txt"

[[arguments]]
name = "name"
type = "string"
description = "Name to greet"
required = true
`)
	writeFile(t, filepath.Join(cmdDir, "run.py"), `import os
print(f"Hello, {os.environ['KONTEKST_ARG_NAME']}!")
`)
	writeFile(t, filepath.Join(cmdDir, "requirements.txt"), "")

	cmd, err := loadCommandDir(cmdDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Runtime != "python" {
		t.Errorf("runtime = %q, want %q", cmd.Runtime, "python")
	}
	if cmd.Timeout != 30 {
		t.Errorf("timeout = %d, want %d", cmd.Timeout, 30)
	}
	if cmd.RequirementsFile != "requirements.txt" {
		t.Errorf("requirements = %q, want %q", cmd.RequirementsFile, "requirements.txt")
	}
}

func TestLoadMissingScriptFile(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "bad")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "bad"
description = "Missing script"
runtime = "bash"
`)

	_, err := loadCommandDir(cmdDir)
	if err == nil {
		t.Fatal("expected error for missing run.sh")
	}
	if !strings.Contains(err.Error(), "run.sh not found") {
		t.Errorf("error = %q, want mention of run.sh", err.Error())
	}
}

func TestLoadInvalidRuntime(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "bad-runtime")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "bad-runtime"
description = "Invalid runtime"
runtime = "ruby"
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), "echo hi")

	_, err := loadCommandDir(cmdDir)
	if err == nil {
		t.Fatal("expected error for invalid runtime")
	}
}

func TestLoadTimeoutDefaults(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "defaults")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "defaults"
description = "Timeout defaults"
runtime = "bash"
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), "echo hi")

	cmd, err := loadCommandDir(cmdDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Timeout != 60 {
		t.Errorf("default timeout = %d, want 60", cmd.Timeout)
	}
}

func TestLoadTimeoutClamped(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "clamped")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "clamped"
description = "Timeout clamped"
runtime = "bash"
timeout = 999
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), "echo hi")

	cmd, err := loadCommandDir(cmdDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Timeout != 300 {
		t.Errorf("clamped timeout = %d, want 300", cmd.Timeout)
	}
}

func TestRegistryLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(filepath.Join(dir, "nonexistent"))

	if err := reg.Load(); err != nil {
		t.Fatalf("unexpected error for nonexistent dir: %v", err)
	}

	if reg.Summaries() != "" {
		t.Error("expected empty summaries for nonexistent dir")
	}
}

func TestRegistryLoadAndGet(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "hello")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "hello"
description = "Says hello"
runtime = "bash"

[[arguments]]
name = "who"
type = "string"
description = "Who to greet"
required = true
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), `echo "Hello $KONTEKST_ARG_WHO"`)

	reg := NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cmd, ok := reg.Get("hello")
	if !ok {
		t.Fatal("Get(hello) returned false")
	}
	if cmd.Name != "hello" {
		t.Errorf("name = %q, want %q", cmd.Name, "hello")
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) returned true")
	}
}

func TestRegistrySummaries(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "greet")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "greet"
description = "Greets someone"
runtime = "bash"

[[arguments]]
name = "name"
type = "string"
description = "Name"
required = true

[[arguments]]
name = "style"
type = "string"
description = "Style"
required = false
default = "casual"
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), "echo hi")

	reg := NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	summaries := reg.Summaries()
	if !strings.Contains(summaries, "greet: Greets someone") {
		t.Errorf("summaries missing command info: %s", summaries)
	}
	if !strings.Contains(summaries, "name (string, required)") {
		t.Errorf("summaries missing required arg: %s", summaries)
	}
	if !strings.Contains(summaries, "style (string, optional") {
		t.Errorf("summaries missing optional arg: %s", summaries)
	}
	if !strings.Contains(summaries, `default: "casual"`) {
		t.Errorf("summaries missing default value: %s", summaries)
	}
}

func TestRegistrySkipsInvalidCommands(t *testing.T) {
	dir := t.TempDir()

	goodDir := filepath.Join(dir, "good")
	writeFile(t, filepath.Join(goodDir, "command.toml"), `
name = "good"
description = "A good command"
runtime = "bash"
`)
	writeFile(t, filepath.Join(goodDir, "run.sh"), "echo good")

	badDir := filepath.Join(dir, "bad")
	writeFile(t, filepath.Join(badDir, "command.toml"), `
name = "bad"
description = "Missing script"
runtime = "bash"
`)

	reg := NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if _, ok := reg.Get("good"); !ok {
		t.Error("good command should be loaded")
	}
	if _, ok := reg.Get("bad"); ok {
		t.Error("bad command should not be loaded")
	}
}

func TestLoadRequirementsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "traversal")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "traversal"
description = "Path traversal attempt"
runtime = "python"

[python]
requirements = "../../etc/evil"
`)
	writeFile(t, filepath.Join(cmdDir, "run.py"), "print('hi')")

	_, err := loadCommandDir(cmdDir)
	if err == nil {
		t.Fatal("expected error for path traversal in requirements")
	}
	if !strings.Contains(err.Error(), "relative path within the command directory") {
		t.Errorf("error = %q, want mention of path safety", err.Error())
	}
}

func TestLoadWorkingDirAgent(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "agent-wd")

	writeFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "agent-wd"
description = "Uses agent working dir"
runtime = "bash"
working_dir = "agent"
`)
	writeFile(t, filepath.Join(cmdDir, "run.sh"), "pwd")

	cmd, err := loadCommandDir(cmdDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.WorkingDir != "agent" {
		t.Errorf("working_dir = %q, want %q", cmd.WorkingDir, "agent")
	}
}
