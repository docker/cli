package credentials

import (
	"github.com/moby/moby/api/types/registry"
)

// Store is the interface that any credentials store must implement.
type Store interface {
	// Erase removes credentials from the store for a given server.
	Erase(serverAddress string) error
	// Get retrieves credentials from the store for a given server.
	Get(serverAddress string) (registry.AuthConfig, error)
	// GetAll retrieves all the credentials from the store.
	GetAll() (map[string]registry.AuthConfig, error)
	// Store saves credentials in the store.
	Store(authConfig registry.AuthConfig) error
}
