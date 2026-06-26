package memorystore

import (
	"testing"

	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestMemoryStore(t *testing.T) {
	config := map[string]types.AuthConfig{
		"https://example.test": {
			Username:      "something-something",
			ServerAddress: "https://example.test",
			Auth:          "super_secret_token",
		},
	}

	fallbackConfig := map[string]types.AuthConfig{
		"https://only-in-file.example.test": {
			Username:      "something-something",
			ServerAddress: "https://only-in-file.example.test",
			Auth:          "super_secret_token",
		},
	}

	fallbackStore, err := New(WithAuthConfig(fallbackConfig))
	assert.NilError(t, err)

	memoryStore, err := New(WithAuthConfig(config), WithFallbackStore(fallbackStore))
	assert.NilError(t, err)

	t.Run("get credentials from memory store", func(t *testing.T) {
		c, err := memoryStore.Get("https://example.test")
		assert.NilError(t, err)
		assert.Equal(t, c, config["https://example.test"])
	})

	t.Run("get credentials from fallback store", func(t *testing.T) {
		c, err := memoryStore.Get("https://only-in-file.example.test")
		assert.NilError(t, err)
		assert.Equal(t, c, fallbackConfig["https://only-in-file.example.test"])
	})

	t.Run("storing credentials in memory store should also be in defined fallback store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "not-in-store",
			ServerAddress: "https://not-in-store.example.test",
			Auth:          "not-in-store_token",
		})
		assert.NilError(t, err)
		c, err := memoryStore.Get("https://not-in-store.example.test")
		assert.NilError(t, err)
		assert.Equal(t, c.Username, "not-in-store")
		assert.Equal(t, c.ServerAddress, "https://not-in-store.example.test")
		assert.Equal(t, c.Auth, "not-in-store_token")

		cc, err := fallbackStore.Get("https://not-in-store.example.test")
		assert.NilError(t, err)
		assert.Equal(t, cc.Username, "not-in-store")
		assert.Equal(t, cc.ServerAddress, "https://not-in-store.example.test")
		assert.Equal(t, cc.Auth, "not-in-store_token")
	})

	t.Run("delete credentials should remove credentials from memory store and fallback store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "a-new-credential",
			ServerAddress: "https://a-new-credential.example.test",
			Auth:          "a-new-credential_token",
		})
		assert.NilError(t, err)
		err = memoryStore.Erase("https://a-new-credential.example.test")
		assert.NilError(t, err)
		_, err = memoryStore.Get("https://a-new-credential.example.test")
		assert.Check(t, is.ErrorIs(err, errValueNotFound))
		_, err = fallbackStore.Get("https://a-new-credential.example.test")
		assert.Check(t, is.ErrorIs(err, errValueNotFound))
	})
}

func TestMemoryStoreWithoutFallback(t *testing.T) {
	config := map[string]types.AuthConfig{
		"https://example.test": {
			Username:      "something-something",
			ServerAddress: "https://example.test",
			Auth:          "super_secret_token",
		},
	}

	memoryStore, err := New(WithAuthConfig(config))
	assert.NilError(t, err)

	t.Run("get credentials from memory store without fallback", func(t *testing.T) {
		c, err := memoryStore.Get("https://example.test")
		assert.NilError(t, err)
		assert.Equal(t, c, config["https://example.test"])
	})

	t.Run("get non-existing credentials from memory store should error", func(t *testing.T) {
		_, err := memoryStore.Get("https://not-in-store.example.test")
		assert.Check(t, is.ErrorIs(err, errValueNotFound))
	})

	t.Run("case store credentials", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "not-in-store",
			ServerAddress: "https://not-in-store.example.test",
			Auth:          "not-in-store_token",
		})
		assert.NilError(t, err)
		c, err := memoryStore.Get("https://not-in-store.example.test")
		assert.NilError(t, err)
		assert.Equal(t, c.Username, "not-in-store")
		assert.Equal(t, c.ServerAddress, "https://not-in-store.example.test")
		assert.Equal(t, c.Auth, "not-in-store_token")
	})

	t.Run("delete credentials should remove credentials from memory store", func(t *testing.T) {
		err := memoryStore.Store(types.AuthConfig{
			Username:      "a-new-credential",
			ServerAddress: "https://a-new-credential.example.test",
			Auth:          "a-new-credential_token",
		})
		assert.NilError(t, err)
		err = memoryStore.Erase("https://a-new-credential.example.test")
		assert.NilError(t, err)
		_, err = memoryStore.Get("https://a-new-credential.example.test")
		assert.Check(t, is.ErrorIs(err, errValueNotFound))
	})
}
