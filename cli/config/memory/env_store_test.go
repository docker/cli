package memory

import (
	"testing"

	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
)

func TestMemoryStore(t *testing.T) {
	config := map[string]types.AuthConfig{
		"https://example.com": {
			Username:      "something-something",
			ServerAddress: "https://example.com",
			Auth:          "super_secret_token",
		},
	}

	fallbackConfig := map[string]types.AuthConfig{
		"https://only-in-file.com": {
			Username:      "something-something",
			ServerAddress: "https://only-in-file.com",
			Auth:          "super_secret_token",
		},
	}

	fallbackStore := NewInMemoryStore(fallbackConfig)

	memoryStore := NewInMemoryStore(config, WithFallbackStore(fallbackStore))

	t.Run("case=get credentials from memory store", func(t *testing.T) {
		c, err := memoryStore.Get("https://example.com")
		assert.NilError(t, err)
		assert.Equal(t, c, config["https://example.com"])
	})

	t.Run("case=get credentials from fallback store", func(t *testing.T) {
		c, err := memoryStore.Get("https://only-in-file.com")
		assert.NilError(t, err)
		assert.Equal(t, c, fallbackConfig["https://only-in-file.com"])
	})

	t.Run("case=storing credentials in memory store should also be in defined fallback store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "not-in-store",
			ServerAddress: "https://not-in-store",
			Auth:          "not-in-store_token",
		})
		assert.NilError(t, err)
		c, err := memoryStore.Get("https://not-in-store")
		assert.NilError(t, err)
		assert.Equal(t, c.Username, "not-in-store")
		assert.Equal(t, c.ServerAddress, "https://not-in-store")
		assert.Equal(t, c.Auth, "not-in-store_token")

		cc, err := fallbackStore.Get("https://not-in-store")
		assert.NilError(t, err)
		assert.Equal(t, cc.Username, "not-in-store")
		assert.Equal(t, cc.ServerAddress, "https://not-in-store")
		assert.Equal(t, cc.Auth, "not-in-store_token")
	})

	t.Run("case=delete credentials should remove credentials from memory store and fallback store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "a-new-credential",
			ServerAddress: "https://a-new-credential",
			Auth:          "a-new-credential_token",
		})
		assert.NilError(t, err)
		err = memoryStore.Erase("https://a-new-credential")
		assert.NilError(t, err)
		_, err = memoryStore.Get("https://a-new-credential")
		assert.Equal(t, IsErrValueNotFound(err), true)
		_, err = fallbackStore.Get("https://a-new-credential")
		assert.Equal(t, IsErrValueNotFound(err), true)
	})
}

func TestMemoryStoreWithoutFallback(t *testing.T) {
	config := map[string]types.AuthConfig{
		"https://example.com": {
			Username:      "something-something",
			ServerAddress: "https://example.com",
			Auth:          "super_secret_token",
		},
	}

	memoryStore := NewInMemoryStore(config)

	t.Run("case=get credentials from memory store without fallback", func(t *testing.T) {
		c, err := memoryStore.Get("https://example.com")
		assert.NilError(t, err)
		assert.Equal(t, c, config["https://example.com"])
	})

	t.Run("case=get non-existing credentials from memory store should error", func(t *testing.T) {
		_, err := memoryStore.Get("https://not-in-store")
		assert.Equal(t, IsErrValueNotFound(err), true)
	})

	t.Run("case store credentials", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "not-in-store",
			ServerAddress: "https://not-in-store",
			Auth:          "not-in-store_token",
		})
		assert.NilError(t, err)
		c, err := memoryStore.Get("https://not-in-store")
		assert.NilError(t, err)
		assert.Equal(t, c.Username, "not-in-store")
		assert.Equal(t, c.ServerAddress, "https://not-in-store")
		assert.Equal(t, c.Auth, "not-in-store_token")
	})

	t.Run("case=delete credentials should remove credentials from memory store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "a-new-credential",
			ServerAddress: "https://a-new-credential",
			Auth:          "a-new-credential_token",
		})
		assert.NilError(t, err)
		err = memoryStore.Erase("https://a-new-credential")
		assert.NilError(t, err)
		_, err = memoryStore.Get("https://a-new-credential")
		assert.Equal(t, IsErrValueNotFound(err), true)
	})
}
