package credentials

import (
	"encoding/json"
	"testing"

	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
)

func TestEnvStore(t *testing.T) {
	envConfig := map[string]types.AuthConfig{
		"https://example.com": {
			Email:         "something-something",
			ServerAddress: "https://example.com",
			Auth:          "super_secret_token",
		},
	}
	d, err := json.Marshal(map[string]map[string]types.AuthConfig{
		"auth": envConfig,
	})
	assert.NilError(t, err)

	t.Setenv("DOCKER_AUTH_CONFIG", string(d))

	fileConfig := map[string]types.AuthConfig{
		"https://only-in-file.com": {
			Email:         "something-something",
			ServerAddress: "https://only-in-file.com",
			Auth:          "super_secret_token",
		},
	}

	var saveCount int
	fileStore := NewFileStore(&fakeStore{
		configs: fileConfig,
		saveFn: func(*fakeStore) error {
			saveCount++
			return nil
		},
	})

	envStore := NewEnvStore(fileStore)

	t.Run("case=get credentials from env", func(t *testing.T) {
		c, err := envStore.Get("https://example.com")
		assert.NilError(t, err)
		assert.Equal(t, c, envConfig["https://example.com"])
	})

	t.Run("case=get credentials from file", func(t *testing.T) {
		c, err := envStore.Get("https://only-in-file.com")
		assert.NilError(t, err)
		assert.Equal(t, c, fileConfig["https://only-in-file.com"])
	})

	t.Run("case=storing credentials should not update env", func(t *testing.T) {
		err := envStore.Store(types.AuthConfig{
			Email:         "not-in-env",
			ServerAddress: "https://not-in-env",
			Auth:          "not-in-env",
		})
		assert.NilError(t, err)
		assert.Equal(t, saveCount, 1)
	})

	t.Run("case=delete credentials should not update env", func(t *testing.T) {
		err := envStore.Erase("https://example.com")
		assert.NilError(t, err)
		c, err := envStore.Get("https://example.com")
		assert.NilError(t, err)
		assert.Equal(t, c, envConfig["https://example.com"])
	})
}
