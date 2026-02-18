package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/commands"
	"github.com/erg0nix/kontekst/internal/config"
	agentConfig "github.com/erg0nix/kontekst/internal/config/agents"
	commandsConfig "github.com/erg0nix/kontekst/internal/config/commands"
	skillsConfig "github.com/erg0nix/kontekst/internal/config/skills"
	"github.com/erg0nix/kontekst/internal/context"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/builtin"
)

type setupResult struct {
	Runner *agent.DefaultRunner
	Agents *agent.Registry
	Skills *skills.Registry
}

func setupServices(cfg config.Config) setupResult {
	if err := agentConfig.EnsureDefaults(cfg.DataDir); err != nil {
		slog.Warn("failed to ensure default agents", "error", err)
	}

	skillsDir := filepath.Join(cfg.DataDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		slog.Warn("failed to create skills directory", "error", err)
	}
	if err := skillsConfig.EnsureDefaults(skillsDir); err != nil {
		slog.Warn("failed to ensure default skills", "error", err)
	}
	skillsRegistry := skills.NewRegistry(skillsDir)
	if err := skillsRegistry.Load(); err != nil {
		slog.Warn("failed to load skills", "error", err)
	}

	commandsDir := filepath.Join(cfg.DataDir, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		slog.Warn("failed to create commands directory", "error", err)
	}
	if err := commandsConfig.EnsureDefaults(commandsDir); err != nil {
		slog.Warn("failed to ensure default commands", "error", err)
	}
	commandsRegistry := commands.NewRegistry(commandsDir)
	if err := commandsRegistry.Load(); err != nil {
		slog.Warn("failed to load commands", "error", err)
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, cfg.DataDir, cfg.Tools)
	builtin.RegisterSkill(toolRegistry, skillsRegistry)
	builtin.RegisterCommand(toolRegistry, commandsRegistry)

	contextService := context.NewFileContextService(cfg.DataDir)
	sessionService := &sessions.FileSessionService{BaseDir: cfg.DataDir}

	runner := &agent.DefaultRunner{
		Tools:       toolRegistry,
		Context:     contextService,
		Sessions:    sessionService,
		DebugConfig: cfg.Debug,
	}

	return setupResult{
		Runner: runner,
		Agents: agent.NewRegistry(cfg.DataDir),
		Skills: skillsRegistry,
	}
}
