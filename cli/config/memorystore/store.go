package memorystore

import (
	"errors"
	"maps"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
)

var errValueNotFound = errors.New("value not found")

func IsErrValueNotFound(err error) bool {
	return errors.Is(err, errValueNotFound)
}

type memoryStore struct {
	memoryCredentials map[string]types.AuthConfig
	fallbackStore     credentials.Store
}

func (e *memoryStore) Erase(serverAddress string) error {
	delete(e.memoryCredentials, serverAddress)

	if e.fallbackStore != nil {
		if err := e.fallbackStore.Erase(serverAddress); err != nil {
			return err
		}
	}

	return nil
}

func (e *memoryStore) Get(serverAddress string) (types.AuthConfig, error) {
	authConfig, ok := e.memoryCredentials[serverAddress]
	if !ok {
		if e.fallbackStore != nil {
			return e.fallbackStore.Get(serverAddress)
		}
		return types.AuthConfig{}, errValueNotFound
	}
	return authConfig, nil
}

func (e *memoryStore) GetAll() (map[string]types.AuthConfig, error) {
	creds := make(map[string]types.AuthConfig)

	if e.fallbackStore != nil {
		fileCredentials, err := e.fallbackStore.GetAll()
		if err == nil {
			creds = fileCredentials
		}
	}

	maps.Copy(creds, e.memoryCredentials)
	return creds, nil
}

func (e *memoryStore) Store(authConfig types.AuthConfig) error {
	e.memoryCredentials[authConfig.ServerAddress] = authConfig

	if e.fallbackStore != nil {
		return e.fallbackStore.Store(authConfig)
	}
	return nil
}

// WithFallbackStore sets a fallback store.
//
// Write opterations will be performed on both the memory store and the
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
func WithFallbackStore(store credentials.Store) func(*memoryStore) {
	return func(s *memoryStore) {
		s.fallbackStore = store
	}
}

// WithAuthConfig allows to set the initial credentials in the memory store.
func WithAuthConfig(config map[string]types.AuthConfig) func(*memoryStore) {
	return func(s *memoryStore) {
		s.memoryCredentials = config
	}
}

// New creates a new in memory credential store
func New(opts ...func(*memoryStore)) credentials.Store {
	m := &memoryStore{
		memoryCredentials: make(map[string]types.AuthConfig),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}
