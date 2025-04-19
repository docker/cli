package memory

import (
	"testing"

	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
)

func TestEnvStore(t *testing.T) {
	config := map[string]types.AuthConfig{
		"https://example.com": {
			Email:         "something-something",
			ServerAddress: "https://example.com",
			Auth:          "super_secret_token",
		},
	}

	fallbackConfig := map[string]types.AuthConfig{
		"https://only-in-file.com": {
			Email:         "something-something",
			ServerAddress: "https://only-in-file.com",
			Auth:          "super_secret_token",
		},
	}

	fallbackStore := NewInMemoryStore(fallbackConfig)

	memoryStore := NewInMemoryStore(config, WithFallbackStore(fallbackStore))

	t.Run("case=get credentials from env", func(t *testing.T) {
		c, err := memoryStore.Get("https://example.com")
		assert.NilError(t, err)
		assert.Equal(t, c, config["https://example.com"])
	})

	t.Run("case=get credentials from file", func(t *testing.T) {
		c, err := memoryStore.Get("https://only-in-file.com")
		assert.NilError(t, err)
		assert.Equal(t, c, fallbackConfig["https://only-in-file.com"])
	})

	t.Run("case=storing credentials should not update env", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Email:         "not-in-env",
			ServerAddress: "https://not-in-env",
			Auth:          "not-in-env",
		})
		assert.NilError(t, err)
	})

	t.Run("case=delete credentials should not update env", func(t *testing.T) {
		err := memoryStore.Erase("https://example.com")
		assert.NilError(t, err)
		_, err = memoryStore.Get("https://example.com")
		assert.Equal(t, IsErrValueNotFound(err), true)
	})
}
