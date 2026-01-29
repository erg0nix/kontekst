package main

import (
	"flag"
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/erg0nix/kontekst/internal/agent"
	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/context"
	grpcsvc "github.com/erg0nix/kontekst/internal/grpc"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
	"github.com/erg0nix/kontekst/internal/providers"
	"github.com/erg0nix/kontekst/internal/sessions"
	"github.com/erg0nix/kontekst/internal/tools"
	"github.com/erg0nix/kontekst/internal/tools/builtin"

	"google.golang.org/grpc"
)

func main() {
	var (
		configPathFlag = flag.String("config", "", "path to config file (default ~/.kontekst/config.toml)")
		bindFlag       = flag.String("bind", "", "gRPC bind address")
		endpointFlag   = flag.String("endpoint", "", "llama-server endpoint")
		modelFlag      = flag.String("model", "", "path to gguf model")
		modelDirFlag   = flag.String("model-dir", "", "directory where models live")
		binFlag        = flag.String("llama-server-bin", "", "llama-server binary")
		dataDirFlag    = flag.String("data-dir", "", "base data dir (default ~/.kontekst)")
	)
	flag.Parse()

	configPath := *configPathFlag
	if configPath == "" {
		configPath = filepath.Join(config.Default().DataDir, "config.toml")
	}

	daemonConfig, err := config.LoadOrCreate(configPath)
	if err != nil {
		log.Fatal(err)
	}

	setIfNotEmpty := func(dst *string, value string) {
		if value != "" {
			*dst = value
		}
	}

	setIfNotEmpty(&daemonConfig.Bind, *bindFlag)
	setIfNotEmpty(&daemonConfig.Endpoint, *endpointFlag)
	setIfNotEmpty(&daemonConfig.Model, *modelFlag)
	setIfNotEmpty(&daemonConfig.ModelDir, *modelDirFlag)
	setIfNotEmpty(&daemonConfig.LlamaServerBin, *binFlag)
	setIfNotEmpty(&daemonConfig.DataDir, *dataDirFlag)

	modelPath := daemonConfig.Model
	if modelPath == "" && daemonConfig.ModelDir != "" {
		modelPath = filepath.Join(daemonConfig.ModelDir, "model.gguf")
	}

	llamaProvider := providers.NewLlamaServerProvider(config.LlamaServerConfig{
		Endpoint:     daemonConfig.Endpoint,
		BinPath:      daemonConfig.LlamaServerBin,
		AutoStart:    true,
		InheritStdio: true,
		ModelPath:    modelPath,
		ContextSize:  daemonConfig.ContextSize,
		GPULayers:    daemonConfig.GPULayers,
		StartupWait:  15 * time.Second,
		HTTPTimeout:  300 * time.Second,
	})
	if modelPath != "" {
		if err := llamaProvider.LoadModel(); err != nil {
			log.Printf("failed to load model: %v", err)
		}
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, daemonConfig.DataDir)

	contextService := &context.FileContextService{
		BaseDir:        daemonConfig.DataDir,
		SystemTemplate: "You are a helpful assistant.",
		UserTemplate:   "{{ user_message }}",
		MaxTokens:      daemonConfig.ContextSize,
	}
	sessionService := &sessions.FileSessionService{BaseDir: daemonConfig.DataDir}
	runService := &sessions.FileRunService{Path: filepath.Join(daemonConfig.DataDir, "runs.jsonl")}

	runner := &agent.AgentRunner{
		Provider: &providers.SingleProviderRouter{Provider: llamaProvider},
		Tools:    toolRegistry,
		Context:  contextService,
		Sessions: sessionService,
		Runs:     runService,
	}

	grpcListener, err := net.Listen("tcp", daemonConfig.Bind)
	if err != nil {
		log.Fatal(err)
	}

	startTime := time.Now()
	grpcServer := grpc.NewServer()

	pb.RegisterAgentServiceServer(grpcServer, &grpcsvc.AgentHandler{Runner: runner})
	pb.RegisterDaemonServiceServer(grpcServer, &grpcsvc.DaemonHandler{
		Config:    daemonConfig,
		Provider:  llamaProvider,
		StartTime: startTime,
		StopFunc:  grpcServer.GracefulStop,
	})

	log.Printf("kontekst-go daemon listening on %s", daemonConfig.Bind)

	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatal(err)
	}
}
