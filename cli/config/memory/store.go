package memory

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
	credentials := make(map[string]types.AuthConfig)

	if e.fallbackStore != nil {
		fileCredentials, err := e.fallbackStore.GetAll()
		if err == nil {
			credentials = fileCredentials
		}
	}

	// override the file credentials with the env credentials
	maps.Copy(credentials, e.memoryCredentials)
	return credentials, nil
}

func (e *memoryStore) Store(authConfig types.AuthConfig) error {
	e.memoryCredentials[authConfig.ServerAddress] = authConfig

	if e.fallbackStore != nil {
		return e.fallbackStore.Store(authConfig)
	}
	return nil
}

func WithFallbackStore(store credentials.Store) func(*memoryStore) {
	return func(s *memoryStore) {
		s.fallbackStore = store
	}
}

// NewInMemoryStore creates a new credentials store
// from config.
//
// Using the `WithFallbackStore` option, it can be configured to
// use a fallback store to retrieve credentials.
func NewInMemoryStore(config map[string]types.AuthConfig, opts ...func(*memoryStore)) credentials.Store {
	m := &memoryStore{
		memoryCredentials: config,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}
