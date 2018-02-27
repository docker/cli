package test

import (
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/docker/api/types"
)

// FakeStore implements a credentials.Store that only acts as an in memory map
type FakeStore struct {
	store      map[string]types.AuthConfig
	eraseFunc  func(serverAddress string) error
	getFunc    func(serverAddress string) (types.AuthConfig, error)
	getAllFunc func() (map[string]types.AuthConfig, error)
	storeFunc  func(authConfig types.AuthConfig) error
}

// NewFakeStore creates a new file credentials store.
func NewFakeStore() credentials.Store {
	return &FakeStore{store: map[string]types.AuthConfig{}}
}

// SetStore is used to overrides Set function
func (c *FakeStore) SetStore(store map[string]types.AuthConfig) {
	c.store = store
}

// SetEraseFunc is used to overrides Erase function
func (c *FakeStore) SetEraseFunc(eraseFunc func(string) error) {
	c.eraseFunc = eraseFunc
}

// SetGetFunc is used to overrides Get function
func (c *FakeStore) SetGetFunc(getFunc func(string) (types.AuthConfig, error)) {
	c.getFunc = getFunc
}

// SetGetAllFunc is used to  overrides GetAll function
func (c *FakeStore) SetGetAllFunc(getAllFunc func() (map[string]types.AuthConfig, error)) {
	c.getAllFunc = getAllFunc
}

// SetStoreFunc is used to override Store function
func (c *FakeStore) SetStoreFunc(storeFunc func(types.AuthConfig) error) {
	c.storeFunc = storeFunc
}

// Erase removes the given credentials from the map store
func (c *FakeStore) Erase(serverAddress string) error {
	if c.eraseFunc != nil {
		return c.eraseFunc(serverAddress)
	}
	delete(c.store, serverAddress)
	return nil
}

// Get retrieves credentials for a specific server from the map store.
func (c *FakeStore) Get(serverAddress string) (types.AuthConfig, error) {
	if c.getFunc != nil {
		return c.getFunc(serverAddress)
	}
	return c.store[serverAddress], nil
}

// GetAll returns the key value pairs of ServerAddress => Username
func (c *FakeStore) GetAll() (map[string]types.AuthConfig, error) {
	if c.getAllFunc != nil {
		return c.getAllFunc()
	}
	return c.store, nil
}

// Store saves the given credentials in the map store.
func (c *FakeStore) Store(authConfig types.AuthConfig) error {
	if c.storeFunc != nil {
		return c.storeFunc(authConfig)
	}
	c.store[authConfig.ServerAddress] = authConfig
	return nil
}
