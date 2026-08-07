package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/config"
	pb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

//go:generate go tool mockgen -destination provider_mock.go -package provider . Provider
type Provider interface {
	HandleMountRequest(ctx context.Context, cfg *config.Config) (*pb.MountResponse, error)
}

type provider struct {
	logger *slog.Logger
}

var _ Provider = (*provider)(nil)

type Option func(*provider)

func WithLogger(l *slog.Logger) Option {
	return func(p *provider) {
		p.logger = l
	}
}

func NewProvider(opts ...Option) *provider {
	p := &provider{
		logger: slog.Default(),
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

func (p *provider) HandleMountRequest(ctx context.Context, cfg *config.Config) (*pb.MountResponse, error) {
	client, err := newScalewayClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create scaleway client: %w", err)
	}

	region, exists := client.GetDefaultRegion()
	if !exists {
		return nil, errors.New("invalid region")
	}

	secretAPI := secret.NewAPI(client)

	files := make([]*pb.File, 0, len(cfg.Secrets))
	objectVersions := make([]*pb.ObjectVersion, 0, len(cfg.Secrets))

	for _, s := range cfg.Secrets {
		var resp *secret.AccessSecretVersionResponse

		if s.ID != "" {
			p.logger.Debug("accessing secret version by id",
				slog.String("region", string(region)),
				slog.String("id", s.ID),
				slog.String("revision", s.Revision),
			)

			resp, err = secretAPI.AccessSecretVersion(&secret.AccessSecretVersionRequest{
				Region:   region,
				SecretID: s.ID,
				Revision: s.Revision,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to access secret version for secret %s: %w", s.ID, err)
			}
		} else {
			p.logger.Debug("accessing secret version by path",
				slog.String("region", string(region)),
				slog.String("projectID", s.ProjectID),
				slog.String("path", s.Path),
				slog.String("name", s.Name),
				slog.String("revision", s.Revision),
			)

			resp, err = secretAPI.AccessSecretVersionByPath(&secret.AccessSecretVersionByPathRequest{
				Region:     region,
				ProjectID:  s.ProjectID,
				SecretPath: s.Path,
				SecretName: s.Name,
				Revision:   s.Revision,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to access secret version for secret at %s/%s: %w", s.Path, s.Name, err)
			}
		}

		files = append(files, &pb.File{
			Path:     s.TargetPath,
			Mode:     int32(cfg.Permission),
			Contents: resp.Data,
		})

		objectVersions = append(objectVersions, &pb.ObjectVersion{
			Id:      resp.SecretID,
			Version: strconv.FormatUint(uint64(resp.Revision), 10),
		})
	}

	return &pb.MountResponse{
		Files:         files,
		ObjectVersion: objectVersions,
	}, nil
}
