package grpc

import (
	"context"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	pb "github.com/erg0nix/kontekst/internal/grpc/pb"
)

type EndpointChecker interface {
	IsHealthy() bool
}

type DaemonHandler struct {
	pb.UnimplementedDaemonServiceServer
	Config    config.Config
	Endpoint  EndpointChecker
	StartTime time.Time
	StopFunc  func()
}

func (h *DaemonHandler) GetStatus(ctx context.Context, _ *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	uptimeSeconds := int64(0)
	if !h.StartTime.IsZero() {
		uptimeSeconds = int64(time.Since(h.StartTime).Seconds())
	}

	startedAtText := ""
	if !h.StartTime.IsZero() {
		startedAtText = h.StartTime.Format(time.RFC3339)
	}

	endpointHealthy := false
	if h.Endpoint != nil {
		endpointHealthy = h.Endpoint.IsHealthy()
	}

	return &pb.GetStatusResponse{
		Bind:             h.Config.Bind,
		Endpoint:         h.Config.Endpoint,
		ModelDir:         h.Config.ModelDir,
		EndpointHealthy:  endpointHealthy,
		UptimeSeconds:    uptimeSeconds,
		StartedAtRfc3339: startedAtText,
		DataDir:          h.Config.DataDir,
	}, nil
}

func (h *DaemonHandler) Shutdown(ctx context.Context, _ *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	if h.StopFunc != nil {
		go h.StopFunc()
	}

	return &pb.ShutdownResponse{Message: "shutting down"}, nil
}
