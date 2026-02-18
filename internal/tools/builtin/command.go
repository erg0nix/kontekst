package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/erg0nix/kontekst/internal/commands"
	"github.com/erg0nix/kontekst/internal/tools"
)

const maxOutputBytes = 128 * 1024

// CommandTool is a tool that executes user-defined commands from the commands registry.
type CommandTool struct {
	Registry *commands.Registry
}

func (tool *CommandTool) Name() string { return "run_command" }

func (tool *CommandTool) Description() string {
	var sb strings.Builder
	sb.WriteString("Executes a user-defined command. Commands are curated scripts that perform actions.\n\n")

	summaries := tool.Registry.Summaries()
	if summaries != "" {
		sb.WriteString("<available_commands>\n")
		sb.WriteString(summaries)
		sb.WriteString("</available_commands>\n\n")
		sb.WriteString("Use the 'name' parameter to select a command and pass values via 'arguments'.")
	} else {
		sb.WriteString("No commands are currently available.")
	}

	return sb.String()
}

func (tool *CommandTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Command name from available_commands list",
			},
			"arguments": map[string]any{
				"type":        "object",
				"description": "Arguments to pass to the command as key-value pairs",
			},
		},
		"required": []string{"name"},
	}
}

func (tool *CommandTool) RequiresApproval() bool { return true }

func (tool *CommandTool) Preview(args map[string]any, ctx context.Context) (string, error) {
	name, _ := getStringArg("name", args)
	if name == "" {
		return "", nil
	}

	cmd, ok := tool.Registry.Get(name)
	if !ok {
		return fmt.Sprintf("Command not found: %s", name), nil
	}

	resolvedArgs := resolveArguments(cmd, args)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Command: %s\n", cmd.Name))
	sb.WriteString(fmt.Sprintf("Runtime: %s\n", cmd.Runtime))
	sb.WriteString(fmt.Sprintf("Script:  %s\n", cmd.ScriptPath))
	sb.WriteString(fmt.Sprintf("Timeout: %ds\n", cmd.Timeout))

	workDir := cmd.Dir
	if cmd.WorkingDir == "agent" {
		if agentDir := tools.WorkingDir(ctx); agentDir != "" {
			workDir = agentDir
		}
	}
	sb.WriteString(fmt.Sprintf("WorkDir: %s\n", workDir))

	if len(resolvedArgs) > 0 {
		sb.WriteString("\nEnvironment:\n")
		for _, a := range cmd.Arguments {
			envName := "KONTEKST_ARG_" + strings.ToUpper(a.Name)
			if val, ok := resolvedArgs[a.Name]; ok {
				sb.WriteString(fmt.Sprintf("  %s=%s\n", envName, val))
			}
		}
	}

	return sb.String(), nil
}

func (tool *CommandTool) Execute(args map[string]any, ctx context.Context) (string, error) {
	name, _ := getStringArg("name", args)
	if name == "" {
		return "", fmt.Errorf("command name is required")
	}

	cmd, ok := tool.Registry.Get(name)
	if !ok {
		return "", fmt.Errorf("command not found: %s", name)
	}

	resolvedArgs, err := validateArguments(cmd, args)
	if err != nil {
		return "", err
	}

	env := buildEnv(cmd, resolvedArgs, ctx)

	if cmd.Runtime == "python" && cmd.RequirementsFile != "" {
		if err := ensureVenv(ctx, cmd); err != nil {
			return "", fmt.Errorf("failed to set up python venv: %w", err)
		}
	}

	interpreter, script := interpreterAndScript(cmd)

	timeout := time.Duration(cmd.Timeout) * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	execCmd := exec.CommandContext(execCtx, interpreter, script)
	execCmd.Dir = resolveWorkDir(cmd, ctx)
	execCmd.Env = env
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	execCmd.Cancel = func() error {
		return syscall.Kill(-execCmd.Process.Pid, syscall.SIGKILL)
	}
	execCmd.WaitDelay = 2 * time.Second

	var output bytes.Buffer
	execCmd.Stdout = &output
	execCmd.Stderr = &output

	err = execCmd.Run()

	result := output.Bytes()
	if len(result) > maxOutputBytes {
		result = result[:maxOutputBytes]
	}

	if err != nil {
		return "", fmt.Errorf("command %q failed: %w\n%s", name, err, string(result))
	}

	return string(result), nil
}

func resolveArguments(cmd *commands.Command, args map[string]any) map[string]string {
	provided := extractArgumentMap(args)
	resolved := make(map[string]string)

	for _, arg := range cmd.Arguments {
		if val, ok := provided[arg.Name]; ok {
			resolved[arg.Name] = val
		} else if arg.Default != "" {
			resolved[arg.Name] = arg.Default
		}
	}

	return resolved
}

func validateArguments(cmd *commands.Command, args map[string]any) (map[string]string, error) {
	resolved := resolveArguments(cmd, args)

	for _, arg := range cmd.Arguments {
		if arg.Required {
			if _, ok := resolved[arg.Name]; !ok {
				return nil, fmt.Errorf("missing required argument: %s", arg.Name)
			}
		}
	}

	return resolved, nil
}

func extractArgumentMap(args map[string]any) map[string]string {
	result := make(map[string]string)
	argsObj, ok := args["arguments"].(map[string]any)
	if !ok {
		return result
	}

	for k, v := range argsObj {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}

	return result
}

var allowedEnvKeys = []string{"PATH", "HOME", "USER", "SHELL", "TMPDIR", "LANG", "TERM"}

func buildEnv(cmd *commands.Command, resolvedArgs map[string]string, ctx context.Context) []string {
	var env []string
	for _, key := range allowedEnvKeys {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}

	env = append(env, "KONTEKST_COMMAND_DIR="+cmd.Dir)

	if agentDir := tools.WorkingDir(ctx); agentDir != "" {
		env = append(env, "KONTEKST_WORKDIR="+agentDir)
	}

	for name, value := range resolvedArgs {
		envName := "KONTEKST_ARG_" + strings.ToUpper(name)
		env = append(env, envName+"="+value)
	}

	return env
}

func interpreterAndScript(cmd *commands.Command) (string, string) {
	switch cmd.Runtime {
	case "python":
		venvPython := filepath.Join(cmd.Dir, ".venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			return venvPython, cmd.ScriptPath
		}
		return "python3", cmd.ScriptPath
	default:
		return "bash", cmd.ScriptPath
	}
}

func resolveWorkDir(cmd *commands.Command, ctx context.Context) string {
	if cmd.WorkingDir == "agent" {
		if agentDir := tools.WorkingDir(ctx); agentDir != "" {
			return agentDir
		}
	}
	return cmd.Dir
}

const venvSetupTimeout = 120 * time.Second

func ensureVenv(ctx context.Context, cmd *commands.Command) error {
	venvDir := filepath.Join(cmd.Dir, ".venv")
	if _, err := os.Stat(venvDir); err == nil {
		return nil
	}

	venvCtx, cancel := context.WithTimeout(ctx, venvSetupTimeout)
	defer cancel()

	createCmd := exec.CommandContext(venvCtx, "python3", "-m", "venv", venvDir)
	createCmd.Dir = cmd.Dir
	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("venv creation failed: %w\n%s", err, string(output))
	}

	reqPath := filepath.Join(cmd.Dir, cmd.RequirementsFile)
	if _, err := os.Stat(reqPath); err != nil {
		return nil
	}

	pip := filepath.Join(venvDir, "bin", "pip")
	installCmd := exec.CommandContext(venvCtx, pip, "install", "-r", reqPath)
	installCmd.Dir = cmd.Dir
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install failed: %w\n%s", err, string(output))
	}

	return nil
}

// RegisterCommand adds the run_command tool to the registry.
func RegisterCommand(registry *tools.Registry, commandsRegistry *commands.Registry) {
	registry.Add(&CommandTool{Registry: commandsRegistry})
}
