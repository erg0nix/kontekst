package main

import (
	"flag"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/commands"
	"github.com/erg0nix/kontekst/internal/config"
	agentcfg "github.com/erg0nix/kontekst/internal/config/agents"
	"github.com/erg0nix/kontekst/internal/context"
	grpcsvc "github.com/erg0nix/kontekst/internal/grpc"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/builtin"

	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	var (
		configPathFlag = flag.String("config", "", "path to config file (default ~/.kontekst/config.toml)")
		bindFlag       = flag.String("bind", "", "gRPC bind address")
		endpointFlag   = flag.String("endpoint", "", "llama-server endpoint")
		modelDirFlag   = flag.String("model-dir", "", "directory where models live")
		dataDirFlag    = flag.String("data-dir", "", "base data dir (default ~/.kontekst)")
	)
	flag.Parse()

	configPath := *configPathFlag
	if configPath == "" {
		configPath = filepath.Join(config.Default().DataDir, "config.toml")
	}

	daemonConfig, err := config.LoadOrCreate(configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setIfNotEmpty := func(dst *string, value string) {
		if value != "" {
			*dst = value
		}
	}

	setIfNotEmpty(&daemonConfig.Bind, *bindFlag)
	setIfNotEmpty(&daemonConfig.Endpoint, *endpointFlag)
	setIfNotEmpty(&daemonConfig.ModelDir, *modelDirFlag)
	setIfNotEmpty(&daemonConfig.DataDir, *dataDirFlag)

	daemonConfig.Debug = config.LoadDebugConfigFromEnv(daemonConfig.Debug)

	if err := agentcfg.EnsureDefault(daemonConfig.DataDir); err != nil {
		logger.Warn("failed to ensure default agent", "error", err)
	}

	llamaProvider := providers.NewLlamaServerProvider(
		config.LlamaServerConfig{
			Endpoint:     daemonConfig.Endpoint,
			AutoStart:    true,
			InheritStdio: true,
			ModelDir:     daemonConfig.ModelDir,
			ContextSize:  daemonConfig.ContextSize,
			GPULayers:    daemonConfig.GPULayers,
			StartupWait:  15 * time.Second,
			HTTPTimeout:  300 * time.Second,
		},
		daemonConfig.Debug,
	)
	if daemonConfig.ModelDir != "" {
		if err := llamaProvider.Start(); err != nil {
			logger.Warn("failed to start llama-server", "error", err)
		}
	}

	skillsDir := filepath.Join(daemonConfig.DataDir, "skills")
	skillsRegistry := skills.NewRegistry(skillsDir)
	if err := skillsRegistry.Load(); err != nil {
		logger.Warn("failed to load skills", "error", err)
	}

	commandsDir := filepath.Join(daemonConfig.DataDir, "commands")
	commandsRegistry := commands.NewRegistry(commandsDir)
	if err := commandsRegistry.Load(); err != nil {
		logger.Warn("failed to load commands", "error", err)
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, daemonConfig.DataDir, daemonConfig.Tools)
	builtin.RegisterSkill(toolRegistry, skillsRegistry)
	builtin.RegisterCommand(toolRegistry, commandsRegistry)

	contextService := context.NewFileContextService(&daemonConfig)
	sessionService := &sessions.FileSessionService{BaseDir: daemonConfig.DataDir}

	runner := &agent.AgentRunner{
		Provider: &providers.SingleProviderRouter{Provider: llamaProvider},
		Tools:    toolRegistry,
		Context:  contextService,
		Sessions: sessionService,
	}

	grpcListener, err := net.Listen("tcp", daemonConfig.Bind)
	if err != nil {
		logger.Error("failed to listen", "address", daemonConfig.Bind, "error", err)
		os.Exit(1)
	}

	startTime := time.Now()
	grpcServer := grpc.NewServer()

	agentRegistry := agent.NewRegistry(daemonConfig.DataDir)
	pb.RegisterAgentServiceServer(grpcServer, &grpcsvc.AgentHandler{Runner: runner, Registry: agentRegistry, Skills: skillsRegistry})
	pb.RegisterDaemonServiceServer(grpcServer, &grpcsvc.DaemonHandler{
		Config:    daemonConfig,
		Provider:  llamaProvider,
		StartTime: startTime,
		StopFunc:  grpcServer.GracefulStop,
	})

	logger.Info("daemon listening", "address", daemonConfig.Bind)

	if err := grpcServer.Serve(grpcListener); err != nil {
		logger.Error("grpc server failed", "error", err)
		os.Exit(1)
	}
}
