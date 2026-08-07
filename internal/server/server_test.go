package server_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/scaleway/scaleway-secrets-store-csi/internal/provider"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	pb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	t.Run("creates server with default logger", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProvider := provider.NewMockProvider(ctrl)
		s := server.NewServer(mockProvider)

		assert.NotNil(t, s)
	})

	t.Run("creates server with custom logger", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProvider := provider.NewMockProvider(ctrl)
		customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		s := server.NewServer(mockProvider, server.WithLogger(customLogger))

		assert.NotNil(t, s)
	})
}

func TestServerVersion(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := provider.NewMockProvider(ctrl)
	s := server.NewServer(mockProvider)

	resp, err := s.Version(context.Background(), &pb.VersionRequest{})
	require.NoError(t, err)

	assert.Equal(t, "v1alpha1", resp.Version)
	assert.Equal(t, "secrets-store-csi-driver-provider-scw", resp.RuntimeName)
}

func TestServerMount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		request       *pb.MountRequest
		setupMock     func(*provider.MockProvider)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful mount request",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par", "objects": "- projectID: proj123\n  secretPath: /my-secrets\n  secretName: db-password\n  revision: latest"}`,
				Secrets:    `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`,
				Permission: "420",
			},
			setupMock: func(mock *provider.MockProvider) {
				mock.EXPECT().HandleMountRequest(gomock.Any(), gomock.Any()).Return(&pb.MountResponse{
					Files: []*pb.File{
						{Path: "db-password", Mode: 0644, Contents: []byte("password123")},
					},
					ObjectVersion: []*pb.ObjectVersion{
						{Id: "secret-123", Version: "1"},
					},
				}, nil)
			},
			expectError: false,
		},
		{
			name: "invalid attributes JSON",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{invalid json}`,
				Secrets:    `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`,
				Permission: "420",
			},
			setupMock:     func(mock *provider.MockProvider) {},
			expectError:   true,
			errorContains: "failed to parse configuration",
		},
		{
			name: "invalid credentials JSON",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par"}`,
				Secrets:    `{invalid json}`,
				Permission: "420",
			},
			setupMock:     func(mock *provider.MockProvider) {},
			expectError:   true,
			errorContains: "failed to parse configuration",
		},
		{
			name: "invalid permission format",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par"}`,
				Secrets:    `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`,
				Permission: "invalid",
			},
			setupMock:     func(mock *provider.MockProvider) {},
			expectError:   true,
			errorContains: "failed to parse configuration",
		},
		{
			name: "provider returns error",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par", "objects": "- projectID: proj123\n  secretPath: /my-secrets\n  secretName: db-password\n  revision: latest"}`,
				Secrets:    `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`,
				Permission: "420",
			},
			setupMock: func(mock *provider.MockProvider) {
				mock.EXPECT().HandleMountRequest(gomock.Any(), gomock.Any()).Return(nil, errors.New("provider error"))
			},
			expectError:   true,
			errorContains: "failed to handle mount request",
		},
		{
			name: "missing required fields in config",
			request: &pb.MountRequest{
				TargetPath: "/mnt/secrets",
				Attributes: `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par", "objects": ""}`,
				Secrets:    `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`,
				Permission: "420",
			},
			setupMock:     func(mock *provider.MockProvider) {},
			expectError:   true,
			errorContains: "failed to parse configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := provider.NewMockProvider(ctrl)
			tt.setupMock(mockProvider)

			s := server.NewServer(mockProvider)
			resp, err := s.Mount(context.Background(), tt.request)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Files)
				assert.NotEmpty(t, resp.ObjectVersion)
			}
		})
	}
}
