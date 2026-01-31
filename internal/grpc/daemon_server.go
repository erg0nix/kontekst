package grpc

import (
	"context"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
	"github.com/erg0nix/kontekst/internal/providers"
)

type DaemonHandler struct {
	pb.UnimplementedDaemonServiceServer
	Config    config.Config
	Provider  *providers.LlamaServerProvider
	StartTime time.Time
	StopFunc  func()
}

func (h *DaemonHandler) GetStatus(ctx context.Context, _ *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	llamaStatus := h.Provider.Status()
	uptimeSeconds := int64(0)

	if !h.StartTime.IsZero() {
		uptimeSeconds = int64(time.Since(h.StartTime).Seconds())
	}
	startedAtText := ""

	if !h.StartTime.IsZero() {
		startedAtText = h.StartTime.Format(time.RFC3339)
	}

	return &pb.GetStatusResponse{
		Bind:               h.Config.Bind,
		Endpoint:           llamaStatus.Endpoint,
		ModelDir:           h.Config.ModelDir,
		LlamaServerHealthy: llamaStatus.Healthy,
		LlamaServerRunning: llamaStatus.Running,
		LlamaServerPid:     int32(llamaStatus.PID),
		UptimeSeconds:      uptimeSeconds,
		StartedAtRfc3339:   startedAtText,
		DataDir:            h.Config.DataDir,
	}, nil
}

func (h *DaemonHandler) Shutdown(ctx context.Context, _ *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	if h.StopFunc != nil {
		go h.StopFunc()
	}

	if h.Provider != nil {
		h.Provider.Stop()
	}

	return &pb.ShutdownResponse{Message: "shutting down"}, nil
}
