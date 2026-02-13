package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/commands"
	"github.com/erg0nix/kontekst/internal/tools"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupBashCommand(t *testing.T, dir, name, script string, manifest string) *commands.Registry {
	t.Helper()
	cmdDir := filepath.Join(dir, name)
	writeTestFile(t, filepath.Join(cmdDir, "command.toml"), manifest)
	writeTestFile(t, filepath.Join(cmdDir, "run.sh"), script)

	reg := commands.NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return reg
}

func setupPythonCommand(t *testing.T, dir, name, script string, manifest string) *commands.Registry {
	t.Helper()
	cmdDir := filepath.Join(dir, name)
	writeTestFile(t, filepath.Join(cmdDir, "command.toml"), manifest)
	writeTestFile(t, filepath.Join(cmdDir, "run.py"), script)

	reg := commands.NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return reg
}

func TestExecuteBashCommand(t *testing.T) {
	dir := t.TempDir()
	reg := setupBashCommand(t, dir, "echo-test", `#!/bin/bash
echo "$KONTEKST_ARG_MESSAGE"`, `
name = "echo-test"
description = "Echoes a message"
runtime = "bash"

[[arguments]]
name = "message"
type = "string"
description = "The message"
required = true
`)

	tool := &CommandTool{Registry: reg}
	result, err := tool.Execute(map[string]any{
		"name":      "echo-test",
		"arguments": map[string]any{"message": "hello world"},
	}, context.Background())

	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if strings.TrimSpace(result) != "hello world" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(result), "hello world")
	}
}

func TestExecutePythonCommand(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	dir := t.TempDir()
	reg := setupPythonCommand(t, dir, "greet", `import os
print(os.environ["KONTEKST_ARG_NAME"])`, `
name = "greet"
description = "Greets by name"
runtime = "python"

[[arguments]]
name = "name"
type = "string"
description = "Name"
required = true
`)

	tool := &CommandTool{Registry: reg}
	result, err := tool.Execute(map[string]any{
		"name":      "greet",
		"arguments": map[string]any{"name": "Alice"},
	}, context.Background())

	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if strings.TrimSpace(result) != "Alice" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(result), "Alice")
	}
}

func TestCommandNotFound(t *testing.T) {
	dir := t.TempDir()
	reg := commands.NewRegistry(dir)
	_ = reg.Load()

	tool := &CommandTool{Registry: reg}
	_, err := tool.Execute(map[string]any{"name": "nonexistent"}, context.Background())

	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("error = %q, want mention of 'command not found'", err.Error())
	}
}

func TestMissingRequiredArgument(t *testing.T) {
	dir := t.TempDir()
	reg := setupBashCommand(t, dir, "needs-arg", "echo hi", `
name = "needs-arg"
description = "Needs argument"
runtime = "bash"

[[arguments]]
name = "required_arg"
type = "string"
description = "Required"
required = true
`)

	tool := &CommandTool{Registry: reg}
	_, err := tool.Execute(map[string]any{"name": "needs-arg"}, context.Background())

	if err == nil {
		t.Fatal("expected error for missing required argument")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("error = %q, want mention of 'missing required argument'", err.Error())
	}
}

func TestDefaultArgumentApplied(t *testing.T) {
	dir := t.TempDir()
	reg := setupBashCommand(t, dir, "with-default", `#!/bin/bash
echo "$KONTEKST_ARG_GREETING"`, `
name = "with-default"
description = "Has default"
runtime = "bash"

[[arguments]]
name = "greeting"
type = "string"
description = "Greeting"
required = false
default = "howdy"
`)

	tool := &CommandTool{Registry: reg}
	result, err := tool.Execute(map[string]any{"name": "with-default"}, context.Background())

	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if strings.TrimSpace(result) != "howdy" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(result), "howdy")
	}
}

func TestTimeoutEnforcement(t *testing.T) {
	dir := t.TempDir()

	cmdDir := filepath.Join(dir, "slow")
	writeTestFile(t, filepath.Join(cmdDir, "command.toml"), `
name = "slow"
description = "Sleeps too long"
runtime = "bash"
timeout = 1
`)
	writeTestFile(t, filepath.Join(cmdDir, "run.sh"), "sleep 60")

	reg := commands.NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	tool := &CommandTool{Registry: reg}
	_, err := tool.Execute(map[string]any{"name": "slow"}, context.Background())

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPreviewOutput(t *testing.T) {
	dir := t.TempDir()
	reg := setupBashCommand(t, dir, "preview-test", "echo hi", `
name = "preview-test"
description = "Preview test"
runtime = "bash"
timeout = 30

[[arguments]]
name = "msg"
type = "string"
description = "Message"
required = true
`)

	tool := &CommandTool{Registry: reg}
	preview, err := tool.Preview(map[string]any{
		"name":      "preview-test",
		"arguments": map[string]any{"msg": "test"},
	}, context.Background())

	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}
	if !strings.Contains(preview, "preview-test") {
		t.Errorf("preview missing command name: %s", preview)
	}
	if !strings.Contains(preview, "bash") {
		t.Errorf("preview missing runtime: %s", preview)
	}
	if !strings.Contains(preview, "run.sh") {
		t.Errorf("preview missing script path: %s", preview)
	}
	if !strings.Contains(preview, "KONTEKST_ARG_MSG=test") {
		t.Errorf("preview missing env var: %s", preview)
	}
}

func TestWorkingDirAgent(t *testing.T) {
	dir := t.TempDir()
	agentDir := t.TempDir()

	reg := setupBashCommand(t, dir, "agent-wd", `#!/bin/bash
pwd`, `
name = "agent-wd"
description = "Uses agent working dir"
runtime = "bash"
working_dir = "agent"
`)

	tool := &CommandTool{Registry: reg}
	ctx := tools.WithWorkingDir(context.Background(), agentDir)
	result, err := tool.Execute(map[string]any{"name": "agent-wd"}, ctx)

	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	gotPath := strings.TrimSpace(result)
	wantPath := agentDir

	gotResolved, err1 := filepath.EvalSymlinks(gotPath)
	wantResolved, err2 := filepath.EvalSymlinks(wantPath)

	if err1 == nil && err2 == nil {
		if gotResolved != wantResolved {
			t.Errorf("working dir = %q, want %q (resolved: %q vs %q)", gotPath, wantPath, gotResolved, wantResolved)
		}
	} else if gotPath != wantPath {
		t.Errorf("working dir = %q, want %q", gotPath, wantPath)
	}
}

func TestEnvVarsSet(t *testing.T) {
	dir := t.TempDir()
	agentDir := t.TempDir()

	reg := setupBashCommand(t, dir, "env-test", `#!/bin/bash
echo "workdir=$KONTEKST_WORKDIR"
echo "cmddir=$KONTEKST_COMMAND_DIR"`, `
name = "env-test"
description = "Env var test"
runtime = "bash"
`)

	tool := &CommandTool{Registry: reg}
	ctx := tools.WithWorkingDir(context.Background(), agentDir)
	result, err := tool.Execute(map[string]any{"name": "env-test"}, ctx)

	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !strings.Contains(result, "workdir="+agentDir) {
		t.Errorf("missing KONTEKST_WORKDIR in output: %s", result)
	}

	cmdDir := filepath.Join(dir, "env-test")
	if !strings.Contains(result, "cmddir="+cmdDir) {
		t.Errorf("missing KONTEKST_COMMAND_DIR in output: %s", result)
	}
}

func TestDescriptionWithCommands(t *testing.T) {
	dir := t.TempDir()
	reg := setupBashCommand(t, dir, "listed", "echo hi", `
name = "listed"
description = "A listed command"
runtime = "bash"
`)

	tool := &CommandTool{Registry: reg}
	desc := tool.Description()

	if !strings.Contains(desc, "<available_commands>") {
		t.Error("description missing available_commands tag")
	}
	if !strings.Contains(desc, "listed: A listed command") {
		t.Errorf("description missing command listing: %s", desc)
	}
}

func TestDescriptionEmpty(t *testing.T) {
	dir := t.TempDir()
	reg := commands.NewRegistry(dir)
	_ = reg.Load()

	tool := &CommandTool{Registry: reg}
	desc := tool.Description()

	if !strings.Contains(desc, "No commands are currently available") {
		t.Errorf("description should indicate no commands: %s", desc)
	}
}
