package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/scaleway/scaleway-secrets-store-csi/internal/config"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/provider"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/version"
	pb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

var _ pb.CSIDriverProviderServer = (*Server)(nil)

// Server implements the secrets-store-csi-driver provider gRPC service interface
type Server struct {
	prov   provider.Provider
	logger *slog.Logger
}

type Option func(*Server)

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

func NewServer(provider provider.Provider, opts ...Option) *Server {
	s := &Server{
		prov:   provider,
		logger: slog.Default(),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Server) Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "secrets-store-csi-driver-provider-scw",
		RuntimeVersion: version.BuildVersion,
	}, nil
}

func (s *Server) Mount(ctx context.Context, req *pb.MountRequest) (*pb.MountResponse, error) {
	s.logger.Debug("received mount request",
		slog.String("attributes", req.GetAttributes()),
		slog.String("targetPath", req.GetTargetPath()),
		slog.String("permission", req.GetPermission()),
	)

	cfg, err := config.Parse(req.GetAttributes(), req.GetSecrets(), req.GetTargetPath(), req.GetPermission())
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	resp, err := s.prov.HandleMountRequest(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to handle mount request: %w", err)
	}

	return resp, nil
}
