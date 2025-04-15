package credentials

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/docker/cli/cli/config/types"
)

type envStore struct {
	envCredentials map[string]types.AuthConfig
	fileStore      Store
}

func (e *envStore) Erase(serverAddress string) error {
	// We cannot erase any credentials in the environment
	// let's fallback to the file store
	if err := e.fileStore.Erase(serverAddress); err != nil {
		return err
	}

	return nil
}

func (e *envStore) Get(serverAddress string) (types.AuthConfig, error) {
	authConfig, ok := e.envCredentials[serverAddress]
	if !ok {
		return e.fileStore.Get(serverAddress)
	}
	return authConfig, nil
}

func (e *envStore) GetAll() (map[string]types.AuthConfig, error) {
	credentials := make(map[string]types.AuthConfig)
	fileCredentials, err := e.fileStore.GetAll()
	if err == nil {
		credentials = fileCredentials
	}

	// override the file credentials with the env credentials
	maps.Copy(credentials, e.envCredentials)
	return credentials, nil
}

func (e *envStore) Store(authConfig types.AuthConfig) error {
	// We cannot store any credentials in the environment
	// let's fallback to the file store
	return e.fileStore.Store(authConfig)
}

// NewEnvStore creates a new credentials store
// from the environment variable DOCKER_AUTH_CONFIG.
// It will parse the value set in the environment variable
// as a JSON object and use it as the credentials store.
//
// As a fallback, it will use the parent store.
// Any parsing errors will be ignored and the parent store
// will be used instead.
func NewEnvStore(parentStore Store) Store {
	v, ok := os.LookupEnv("DOCKER_AUTH_CONFIG")
	if !ok {
		return parentStore
	}

	var credentials map[string]map[string]types.AuthConfig
	if err := json.Unmarshal([]byte(v), &credentials); err != nil {
		return parentStore
	}
	auth, ok := credentials["auth"]
	if !ok {
		return parentStore
	}

	return &envStore{
		envCredentials: auth,
		fileStore:      parentStore,
	}
}
