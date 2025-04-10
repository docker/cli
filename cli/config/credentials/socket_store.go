package credentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/docker/cli/cli/config/types"
)

type socketStore struct {
	socketPath string
	client     http.Client
}

// Erase implements Store.
func (s *socketStore) Erase(serverAddress string) error {
	q := url.Values{"key": {serverAddress}}
	req, err := http.NewRequest(http.MethodDelete, "http://localhost/credentials?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to erase credentials")
	}
	return nil
}

// Get implements Store.
func (s *socketStore) Get(serverAddress string) (types.AuthConfig, error) {
	q := url.Values{"key": {serverAddress}}
	req, err := http.NewRequest(http.MethodGet, "http://localhost/credentials?"+q.Encode(), nil)
	if err != nil {
		return types.AuthConfig{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return types.AuthConfig{}, err
	}
	defer resp.Body.Close()

	var authConfig types.AuthConfig
	if err := json.NewDecoder(resp.Body).Decode(&authConfig); err != nil {
		return types.AuthConfig{}, err
	}
	return authConfig, nil
}

// GetAll implements Store.
func (s *socketStore) GetAll() (map[string]types.AuthConfig, error) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost/credentials", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var authConfigs map[string]types.AuthConfig
	if err := json.NewDecoder(resp.Body).Decode(&authConfigs); err != nil {
		return nil, err
	}
	return authConfigs, nil
}

// Store implements Store.
func (s *socketStore) Store(authConfig types.AuthConfig) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(authConfig); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost/credentials", &buf)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to store credentials")
	}
	return nil
}

func NewSocketStore(socketPath string) Store {
	return &socketStore{
		socketPath: socketPath,
		client: http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}
