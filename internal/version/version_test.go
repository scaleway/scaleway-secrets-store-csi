package version_test

import (
	"testing"

	"github.com/scaleway/scaleway-secrets-store-csi/internal/version"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("returns valid version with all fields", func(t *testing.T) {
		t.Parallel()

		version.BuildVersion = "v1.2.3"
		version.BuildDate = "2024-01-15T10:30:00Z"
		version.GoVersion = "go1.21.5"

		v := version.Get()

		assert.Equal(t, "v1.2.3", v.Version)
		assert.Equal(t, "2024-01-15T10:30:00Z", v.BuildDate)
		assert.Equal(t, "go1.21.5", v.GoVersion)
		assert.Equal(t, "0.1.0", v.MinDriverVersion)
	})
}
