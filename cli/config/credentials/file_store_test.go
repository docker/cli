package credentials

import (
	"testing"

	"github.com/docker/cli/cli/config/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeStore struct {
	configs map[string]types.AuthConfig
	saveFn  func(*fakeStore) error
}

func (f *fakeStore) Save() error {
	if f.saveFn != nil {
		// Pass a reference to the fakeStore itself in case saveFn
		// wants to access it.
		return f.saveFn(f)
	}
	return nil
}

func (f *fakeStore) GetAuthConfigs() map[string]types.AuthConfig {
	return f.configs
}

func (f *fakeStore) GetFilename() string {
	return "no-config.json"
}

// TestFileStoreIdempotent verifies that the config-file isn't updated
// if nothing changed.
func TestFileStoreIdempotent(t *testing.T) {
	var saveCount, expectedSaveCount int

	s := NewFileStore(&fakeStore{
		configs: map[string]types.AuthConfig{},
		saveFn: func(*fakeStore) error {
			saveCount++
			return nil
		},
	})
	authOne := types.AuthConfig{
		Auth:          "super_secret_token",
		Email:         "foo@example.com",
		ServerAddress: "https://example.com",
	}
	authTwo := types.AuthConfig{
		Auth:          "also_super_secret_token",
		Email:         "bar@example.com",
		ServerAddress: "https://other.example.com",
	}

	expectedSaveCount = 1
	t.Run("store new credentials", func(t *testing.T) {
		assert.NilError(t, s.Store(authOne))
		retrievedAuth, err := s.Get(authOne.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, authOne))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount))
	})
	t.Run("store same credentials is a no-op", func(t *testing.T) {
		assert.NilError(t, s.Store(authOne))
		retrievedAuth, err := s.Get(authOne.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, authOne))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount), "should not have saved if nothing changed")
	})
	t.Run("store other credentials", func(t *testing.T) {
		expectedSaveCount++
		assert.NilError(t, s.Store(authTwo))
		retrievedAuth, err := s.Get(authTwo.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, authTwo))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount))
	})
	t.Run("erase credentials", func(t *testing.T) {
		expectedSaveCount++
		assert.NilError(t, s.Erase(authOne.ServerAddress))
		retrievedAuth, err := s.Get(authOne.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, types.AuthConfig{}))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount))
	})
	t.Run("erase non-existing credentials is a no-op", func(t *testing.T) {
		assert.NilError(t, s.Erase(authOne.ServerAddress))
		retrievedAuth, err := s.Get(authOne.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, types.AuthConfig{}))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount), "should not have saved if nothing changed")
	})
	t.Run("erase other credentials", func(t *testing.T) {
		expectedSaveCount++
		assert.NilError(t, s.Erase(authTwo.ServerAddress))
		retrievedAuth, err := s.Get(authTwo.ServerAddress)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(retrievedAuth, types.AuthConfig{}))
		assert.Check(t, is.Equal(saveCount, expectedSaveCount))
	})
}

func TestFileStoreAddCredentials(t *testing.T) {
	f := &fakeStore{configs: map[string]types.AuthConfig{}}

	s := NewFileStore(f)
	auth := types.AuthConfig{
		Auth:          "super_secret_token",
		Email:         "foo@example.com",
		ServerAddress: "https://example.com",
	}
	err := s.Store(auth)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 1))

	actual, ok := f.GetAuthConfigs()["https://example.com"]
	assert.Check(t, ok)
	assert.Check(t, is.DeepEqual(auth, actual))
}

func TestFileStoreGet(t *testing.T) {
	f := &fakeStore{configs: map[string]types.AuthConfig{
		"https://example.com": {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
	}}

	s := NewFileStore(f)
	a, err := s.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if a.Auth != "super_secret_token" {
		t.Fatalf("expected auth `super_secret_token`, got %s", a.Auth)
	}
	if a.Email != "foo@example.com" {
		t.Fatalf("expected email `foo@example.com`, got %s", a.Email)
	}
}

func TestFileStoreGetAll(t *testing.T) {
	s1 := "https://example.com"
	s2 := "https://example2.example.com"
	f := &fakeStore{configs: map[string]types.AuthConfig{
		s1: {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
		s2: {
			Auth:          "super_secret_token2",
			Email:         "foo@example2.com",
			ServerAddress: "https://example2.example.com",
		},
	}}

	s := NewFileStore(f)
	as, err := s.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 2 {
		t.Fatalf("wanted 2, got %d", len(as))
	}
	if as[s1].Auth != "super_secret_token" {
		t.Fatalf("expected auth `super_secret_token`, got %s", as[s1].Auth)
	}
	if as[s1].Email != "foo@example.com" {
		t.Fatalf("expected email `foo@example.com`, got %s", as[s1].Email)
	}
	if as[s2].Auth != "super_secret_token2" {
		t.Fatalf("expected auth `super_secret_token2`, got %s", as[s2].Auth)
	}
	if as[s2].Email != "foo@example2.com" {
		t.Fatalf("expected email `foo@example2.com`, got %s", as[s2].Email)
	}
}

func TestFileStoreErase(t *testing.T) {
	f := &fakeStore{configs: map[string]types.AuthConfig{
		"https://example.com": {
			Auth:          "super_secret_token",
			Email:         "foo@example.com",
			ServerAddress: "https://example.com",
		},
	}}

	s := NewFileStore(f)
	err := s.Erase("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	// file store never returns errors, check that the auth config is empty
	a, err := s.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if a.Auth != "" {
		t.Fatalf("expected empty auth token, got %s", a.Auth)
	}
	if a.Email != "" {
		t.Fatalf("expected empty email, got %s", a.Email)
	}
}

func TestConvertToHostname(t *testing.T) {
	tests := []struct{ input, expected string }{
		{
			input:    "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			input:    "::1",
			expected: "::1",
		},
		{
			// FIXME(thaJeztah): this should be normalized to "::1" if there's no port (or vice-versa, as long as we're consistent)
			input:    "[::1]",
			expected: "[::1]",
		},
		{
			input:    "example.com",
			expected: "example.com",
		},
		{
			input:    "http://example.com",
			expected: "example.com",
		},
		{
			input:    "https://example.com",
			expected: "example.com",
		},
		{
			input:    "https://example.com/",
			expected: "example.com",
		},
		{
			input:    "https://example.com/v2/",
			expected: "example.com",
		},
		{
			// FIXME(thaJeztah): should ConvertToHostname correctly handle this / fail on this?
			input:    "unix:///var/run/docker.sock",
			expected: "unix:",
		},
		{
			// FIXME(thaJeztah): should ConvertToHostname correctly handle this?
			input:    "ftp://example.com",
			expected: "example.com",
		},
		// should support non-standard port in registry url
		{
			input:    "127.0.0.1:6556",
			expected: "127.0.0.1:6556",
		},
		{
			// FIXME(thaJeztah): this should be normalized to "[::1]:6556"
			input:    "::1:6556",
			expected: "::1:6556",
		},
		{
			input:    "[::1]:6556",
			expected: "[::1]:6556",
		},
		{
			input:    "example.com:6555",
			expected: "example.com:6555",
		},
		{
			input:    "https://127.0.0.1:6555/v2/",
			expected: "127.0.0.1:6555",
		},
		{
			input:    "https://::1:6555/v2/",
			expected: "[::1]:6555",
		},
		{
			input:    "https://[::1]:6555/v2/",
			expected: "[::1]:6555",
		},
		{
			input:    "http://example.com:6555",
			expected: "example.com:6555",
		},
		{
			input:    "https://example.com:6555",
			expected: "example.com:6555",
		},
		{
			input:    "https://example.com:6555/v2/",
			expected: "example.com:6555",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			actual := ConvertToHostname(tc.input)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
