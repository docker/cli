package credentials

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/credentials/internal/oauth/manager"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/oauth"
	"github.com/docker/docker/registry"
)

// oauthStore wraps an existing store that transparently handles oauth
// flows, managing authentication/token refresh and piggybacking off an
// existing store for storage/retrieval.
type oauthStore struct {
	backingStore Store
	manager      oauth.Manager
}

// NewOAuthStore creates a new oauthStore backed by the provided store.
func NewOAuthStore(backingStore Store) Store {
	m, _ := manager.NewManager()
	return &oauthStore{
		backingStore: backingStore,
		manager:      m,
	}
}

const minimumTokenLifetime = 50 * time.Minute

// todo(laurazard): maybe don't start the login flow here by default,
// and instead only parse/refresh. Leave the login flow to the caller.

// Get retrieves the credentials from the backing store, refreshing the
// access token if the retrieved token is valid for less than 50 minutes.
// If there are no credentials in the backing store, the device code flow
// is initiated with the tenant in order to log the user in and get
func (c *oauthStore) Get(serverAddress string) (types.AuthConfig, error) {
	if serverAddress != registry.IndexServer {
		return c.backingStore.Get(serverAddress)
	}

	auth, err := c.backingStore.Get(serverAddress)
	if err != nil {
		// If an error happens here, it's not due to the backing store not
		// containing credentials, but rather an actual issue with the backing
		// store itself. This should be propagated up.
		return types.AuthConfig{}, err
	}
	tokenRes, err := c.parseToken(auth.Password)
	if err != nil && auth.Password != "" {
		return types.AuthConfig{
			Username:      auth.Username,
			Password:      auth.Password,
			Email:         auth.Email,
			ServerAddress: registry.IndexServer,
		}, nil
	}

	var failedRefresh bool
	// if the access token is valid for less than 50 minutes, refresh it
	if tokenRes.RefreshToken != "" && tokenRes.Claims.Expiry.Time().Before(time.Now().Add(minimumTokenLifetime)) {
		refreshRes, err := c.manager.RefreshToken(context.TODO(), tokenRes.RefreshToken)
		if err != nil {
			failedRefresh = true
		}
		tokenRes = refreshRes
	}

	if tokenRes.AccessToken == "" || failedRefresh {
		tokenRes, err = c.manager.LoginDevice(context.TODO(), os.Stderr)
		if err != nil {
			return types.AuthConfig{}, err
		}
	}

	err = c.storeInBackingStore(tokenRes)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      tokenRes.Claims.Domain.Username,
		Password:      tokenRes.AccessToken,
		Email:         tokenRes.Claims.Domain.Email,
		ServerAddress: registry.IndexServer,
	}, nil
}

// GetAll returns a map containing solely the auth config for the official
// registry, parsed from the backing store and refreshed if necessary.
func (c *oauthStore) GetAll() (map[string]types.AuthConfig, error) {
	allAuths, err := c.backingStore.GetAll()
	if err != nil {
		return nil, err
	}

	if _, ok := allAuths[registry.IndexServer]; !ok {
		return allAuths, nil
	}

	auth, err := c.Get(registry.IndexServer)
	if err != nil {
		return nil, err
	}
	allAuths[registry.IndexServer] = auth
	return allAuths, err
}

// Erase removes the credentials from the backing store, logging out of the
// tenant if running
func (c *oauthStore) Erase(serverAddress string) error {
	if serverAddress == registry.IndexServer {
		// todo(laurazard): should this log out from the tenant
		_ = c.manager.Logout(context.TODO())
	}
	return c.backingStore.Erase(serverAddress)
}

// Store stores the provided credentials in the backing credential store,
// except when the credentials are for the official registry, in which case
// no action is taken because the credentials retrieved/stored during Get.
func (c *oauthStore) Store(auth types.AuthConfig) error {
	if auth.ServerAddress != registry.IndexServer {
		return c.backingStore.Store(auth)
	}

	_, err := c.parseToken(auth.Password)
	if err == nil {
		return nil
	}

	return c.backingStore.Store(auth)
}

func (c *oauthStore) parseToken(password string) (oauth.TokenResult, error) {
	parts := strings.Split(password, "..")
	if len(parts) != 2 {
		return oauth.TokenResult{}, errors.New("failed to parse token")
	}
	accessToken := parts[0]
	refreshToken := parts[1]
	claims, err := oauth.GetClaims(parts[0])
	if err != nil {
		return oauth.TokenResult{}, err
	}
	return oauth.TokenResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Claims:       claims,
	}, nil
}

func (c *oauthStore) storeInBackingStore(tokenRes oauth.TokenResult) error {
	auth := types.AuthConfig{
		Username:      tokenRes.Claims.Domain.Username,
		Password:      c.concat(tokenRes.AccessToken, tokenRes.RefreshToken),
		Email:         tokenRes.Claims.Domain.Email,
		ServerAddress: registry.IndexServer,
	}
	return c.backingStore.Store(auth)
}

func (c *oauthStore) concat(accessToken, refreshToken string) string {
	return accessToken + ".." + refreshToken
}
