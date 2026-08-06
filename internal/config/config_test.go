package config_test

import (
	"testing"

	"github.com/scaleway/secrets-store-csi-driver-provider-scw/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("valid configuration with secrets by path", func(t *testing.T) {
		t.Parallel()

		attributes := `{
				"apiURL": "https://api.scaleway.com",
				"defaultRegion": "fr-par",
				"objects": "- projectID: proj123\n  secretPath: /my-secrets\n  secretName: db-password\n  revision: latest"
			}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		assert.Equal(t, "/mnt/secrets", cfg.TargetPath)
	})

	t.Run("valid configuration with secrets by ID", func(t *testing.T) {
		t.Parallel()

		attributes := `{
				"apiURL": "https://api.scaleway.com",
				"defaultRegion": "fr-par",
				"objects": "- secretID: secret-123\n  revision: latest\n  targetPath: my-secret"
			}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		assert.Equal(t, "/mnt/secrets", cfg.TargetPath)
	})

	t.Run("invalid attributes JSON", func(t *testing.T) {
		t.Parallel()

		attributes := `{invalid json}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse attributes")
	})

	t.Run("invalid credentials JSON", func(t *testing.T) {
		t.Parallel()

		attributes := `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par"}`
		credentials := `{invalid json}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse credentials")
	})

	t.Run("missing target path", func(t *testing.T) {
		t.Parallel()

		attributes := `{
				"apiURL": "https://api.scaleway.com",
				"defaultRegion": "fr-par"
			}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing target path")
	})

	t.Run("invalid permission format", func(t *testing.T) {
		t.Parallel()

		attributes := `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par"}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal permission")
	})
}

func TestParseSecrets(t *testing.T) {
	t.Parallel()

	t.Run("auto-generates targetPath when not specified", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /my-secrets\\n  secretName: db-password\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)

		assert.Equal(t, "my-secrets/db-password", cfg.Secrets[0].TargetPath)
	})

	t.Run("uses custom targetPath when specified", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /my-secrets\\n  secretName: db-password\\n  revision: latest\\n  targetPath: custom-path\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)

		assert.Equal(t, "custom-path", cfg.Secrets[0].TargetPath)
	})

	t.Run("cleans targetPath", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /my-secrets\\n  secretName: db-password\\n  revision: latest\\n  targetPath: ./custom/../path\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)

		assert.Equal(t, "path", cfg.Secrets[0].TargetPath)
	})

	t.Run("missing revision defaults to latest_enabled", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /my-secrets\\n  secretName: pwd\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)
		assert.Equal(t, "latest_enabled", cfg.Secrets[0].Revision)
	})

	t.Run("no secrets specified", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no secret specified")
	})

	t.Run("absolute path in secretPath", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /absolute/path\\n  secretName: pwd\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		assert.Equal(t, "/mnt/secrets", cfg.TargetPath)
	})

	t.Run("absolute path in targetPath", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- projectID: proj123\\n  secretPath: /my-secrets\\n  secretName: pwd\\n  revision: latest\\n  targetPath: /absolute/target\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target path must be relative")
	})

	t.Run("missing projectID for path-based secret uses defaultProjectID", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"defaultProjectID\": \"proj-123\", \"objects\": \"- secretPath: /my-secrets\\n  secretName: pwd\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)
		assert.Equal(t, "proj-123", cfg.Secrets[0].ProjectID)
	})

	t.Run("projectID takes precedence over defaultProjectID", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"defaultProjectID\": \"proj-123\", \"objects\": \"- projectID: custom-proj\\n  secretPath: /my-secrets\\n  secretName: pwd\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		cfg, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.NoError(t, err)
		require.Len(t, cfg.Secrets, 1)
		assert.Equal(t, "custom-proj", cfg.Secrets[0].ProjectID)
	})

	t.Run("missing targetPath for ID-based secret", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- secretID: secret-123\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing target path")
	})

	t.Run("both ID and path specified", func(t *testing.T) {
		t.Parallel()

		attributes := "{\"apiURL\": \"https://api.scaleway.com\", \"defaultRegion\": \"fr-par\", \"objects\": \"- secretID: secret-123\\n  projectID: proj123\\n  secretPath: my-secrets\\n  secretName: pwd\\n  revision: latest\"}"
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret can be specified by id or path but not both")
	})
}

func TestAttributesValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid attributes", func(t *testing.T) {
		t.Parallel()

		attributes := config.Attributes{
			APIURL:                "https://api.scaleway.com",
			DefaultRegion:         "fr-par",
			DefaultOrganizationID: "org-123",
			DefaultProjectID:      "proj-123",
		}

		err := attributes.Validate()
		require.NoError(t, err)
	})

	t.Run("missing API URL", func(t *testing.T) {
		t.Parallel()

		attributes := config.Attributes{
			DefaultRegion: "fr-par",
		}

		err := attributes.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing api url")
	})

	t.Run("missing default region", func(t *testing.T) {
		t.Parallel()

		attributes := config.Attributes{
			APIURL: "https://api.scaleway.com",
		}

		err := attributes.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing default region")
	})

	t.Run("invalid region", func(t *testing.T) {
		t.Parallel()

		attributes := config.Attributes{
			APIURL:        "https://api.scaleway.com",
			DefaultRegion: "invalid-region",
		}

		err := attributes.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "region does not exist")
	})

	t.Run("invalid attributes JSON", func(t *testing.T) {
		t.Parallel()

		attributes := `{invalid json}`
		credentials := `{"accessKey": "SCW1234567890", "secretKey": "secret-key-123"}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse attributes")
	})
}

func TestCredentialsValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid credentials", func(t *testing.T) {
		t.Parallel()

		credentials := config.Credentials{
			AccessKey: "SCW1234567890",
			SecretKey: "secret-key-123",
		}

		err := credentials.Validate()
		require.NoError(t, err)
	})

	t.Run("missing access key", func(t *testing.T) {
		t.Parallel()

		credentials := config.Credentials{
			SecretKey: "secret-key-123",
		}

		err := credentials.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing access key")
	})

	t.Run("missing secret key", func(t *testing.T) {
		t.Parallel()

		credentials := config.Credentials{
			AccessKey: "SCW1234567890",
		}

		err := credentials.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing secret key")
	})

	t.Run("missing both keys", func(t *testing.T) {
		t.Parallel()

		credentials := config.Credentials{}

		err := credentials.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing access key")
	})

	t.Run("invalid credentials JSON", func(t *testing.T) {
		t.Parallel()

		attributes := `{"apiURL": "https://api.scaleway.com", "defaultRegion": "fr-par"}`
		credentials := `{invalid json}`

		_, err := config.Parse(attributes, credentials, "/mnt/secrets", "420")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse credentials")
	})
}

func TestSecretValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		secret        config.Secret
		expectError   bool
		errorContains string
	}{
		{
			name: "path-based secret missing path",
			secret: config.Secret{
				ProjectID:  "proj-123",
				Name:       "db-password",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "missing secret path or secret name",
		},
		{
			name: "valid secret by ID",
			secret: config.Secret{
				ID:         "secret-123",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError: false,
		},
		{
			name: "missing revision",
			secret: config.Secret{
				ProjectID:  "proj-123",
				Path:       "/secrets",
				Name:       "db-password",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "missing revision",
		},
		{
			name: "both ID and path specified",
			secret: config.Secret{
				ID:         "secret-123",
				ProjectID:  "proj-123",
				Path:       "/secrets",
				Name:       "db-password",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "secret can be specified by id or path but not both",
		},
		{
			name: "ID-based secret missing targetPath",
			secret: config.Secret{
				ID:       "secret-123",
				Revision: "latest",
			},
			expectError:   true,
			errorContains: "missing target path",
		},
		{
			name: "path-based secret missing projectID",
			secret: config.Secret{
				Path:       "/secrets",
				Name:       "db-password",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "missing project id",
		},
		{
			name: "path-based secret missing path",
			secret: config.Secret{
				ProjectID:  "proj-123",
				Name:       "db-password",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "missing secret path or secret name",
		},
		{
			name: "path-based secret missing name",
			secret: config.Secret{
				ProjectID:  "proj-123",
				Path:       "/secrets",
				Revision:   "latest",
				TargetPath: "my-secret",
			},
			expectError:   true,
			errorContains: "missing secret path or secret name",
		},
		{
			name: "absolute path in targetPath",
			secret: config.Secret{
				ProjectID:  "proj-123",
				Path:       "/secrets",
				Name:       "db-password",
				Revision:   "latest",
				TargetPath: "/absolute/target",
			},
			expectError:   true,
			errorContains: "target path must be relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.secret.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
