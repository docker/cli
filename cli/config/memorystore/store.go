// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

package memorystore

import (
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
)

// notFoundErr is the error returned when a plugin could not be found.
type notFoundErr string

func (notFoundErr) NotFound() {}

func (e notFoundErr) Error() string {
	return string(e)
}

var errValueNotFound notFoundErr = "value not found"

type Config struct {
	lock              sync.RWMutex
	memoryCredentials map[string]types.AuthConfig
	fallbackStore     credentials.Store
	preferFallback    bool
}

func (e *Config) Erase(serverAddress string) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.memoryCredentials, serverAddress)

	if e.fallbackStore != nil {
		err := e.fallbackStore.Erase(serverAddress)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "memorystore: ", err)
		}
	}

	return nil
}

func (e *Config) Get(serverAddress string) (types.AuthConfig, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.preferFallback && e.fallbackStore != nil {
		if authConfig, err := e.fallbackStore.Get(serverAddress); err == nil && hasAuthConfig(authConfig) {
			return authConfig, nil
		}
	}

	authConfig, ok := e.memoryCredentials[serverAddress]
	if !ok {
		if e.fallbackStore != nil {
			return e.fallbackStore.Get(serverAddress)
		}
		return types.AuthConfig{}, errValueNotFound
	}
	return authConfig, nil
}

func (e *Config) GetAll() (map[string]types.AuthConfig, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	creds := make(map[string]types.AuthConfig)

	if e.preferFallback {
		maps.Copy(creds, e.memoryCredentials)
	}

	if e.fallbackStore != nil {
		fileCredentials, err := e.fallbackStore.GetAll()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "memorystore: ", err)
		} else {
			copyAuthConfigs(creds, fileCredentials, e.preferFallback)
		}
	}

	if !e.preferFallback {
		maps.Copy(creds, e.memoryCredentials)
	}
	return creds, nil
}

func (e *Config) Store(authConfig types.AuthConfig) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.memoryCredentials[authConfig.ServerAddress] = authConfig

	if e.fallbackStore != nil {
		return e.fallbackStore.Store(authConfig)
	}
	return nil
}

func hasAuthConfig(authConfig types.AuthConfig) bool {
	return authConfig.Username != "" ||
		authConfig.Password != "" ||
		authConfig.Auth != "" ||
		authConfig.IdentityToken != "" ||
		authConfig.RegistryToken != ""
}

func copyAuthConfigs(dst, src map[string]types.AuthConfig, skipEmpty bool) {
	for serverAddress, authConfig := range src {
		if skipEmpty && !hasAuthConfig(authConfig) {
			continue
		}
		dst[serverAddress] = authConfig
	}
}

// WithFallbackStore sets a fallback store.
//
// Write operations will be performed on both the memory store and the
// fallback store.
//
// Read operations will first check the memory store, and if the credential
// is not found, it will then check the fallback store.
//
// Retrieving all credentials will return from both the memory store and the
// fallback store, merging the results from both stores into a single map.
//
// Data stored in the memory store will take precedence over data in the
// fallback store.
func WithFallbackStore(store credentials.Store) Options {
	return func(s *Config) error {
		s.fallbackStore = store
		return nil
	}
}

// WithPreferFallback configures the store to prefer reading credentials from
// the fallback store. Write operations are still performed on both stores.
func WithPreferFallback() Options {
	return func(s *Config) error {
		s.preferFallback = true
		return nil
	}
}

// WithAuthConfig allows to set the initial credentials in the memory store.
func WithAuthConfig(config map[string]types.AuthConfig) Options {
	return func(s *Config) error {
		s.memoryCredentials = config
		return nil
	}
}

type Options func(*Config) error

// New creates a new in memory credential store
func New(opts ...Options) (credentials.Store, error) {
	m := &Config{
		memoryCredentials: make(map[string]types.AuthConfig),
	}
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	return m, nil
}
