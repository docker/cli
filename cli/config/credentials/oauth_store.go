package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/oauth"
)

// oauthStore wraps an existing store that transparently handles oauth
// flows, managing authentication/token refresh and piggybacking off an
// existing store for storage/retrieval.
type oauthStore struct {
	backingStore Store
	manager      oauth.Manager
}

// NewOAuthStore creates a new oauthStore backed by the provided store.
func NewOAuthStore(backingStore Store, manager oauth.Manager) Store {
	return &oauthStore{
		backingStore: backingStore,
		manager:      manager,
	}
}

const minimumTokenLifetime = 50 * time.Minute

// see https://github.com/moby/buildkit/pull/5165#discussion_r1682531996
const defaultRegistry = "https://index.docker.io/v1/"

// Get retrieves the credentials from the backing store, refreshing the
// access token if the stored credentials are valid for less than minimumTokenLifetime.
// If the credentials being retrieved are not for the official registry, they are
// returned as is. If the credentials retrieved do not parse as a token, they are
// also returned as is.
func (o *oauthStore) Get(serverAddress string) (types.AuthConfig, error) {
	if serverAddress != defaultRegistry {
		return o.backingStore.Get(serverAddress)
	}

	tokenRes := o.fetchFromBackingStore()
	if tokenRes == nil {
		return o.backingStore.Get(serverAddress)
	}

	// if the access token is valid for less than minimumTokenLifetime, refresh it
	if tokenRes.RefreshToken != "" &&
		(tokenRes.Claims.Expiry == nil ||
			tokenRes.Claims.Expiry.Time().Before(time.Now().Add(minimumTokenLifetime))) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		refreshRes, err := o.manager.RefreshToken(ctx, tokenRes.RefreshToken)
		if err != nil || refreshRes == nil {
			return types.AuthConfig{}, fmt.Errorf("failed to refresh token: %w", err)
		}
		tokenRes = refreshRes
	}

	err := o.storeInBackingStore(*tokenRes)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      tokenRes.Claims.Domain.Username,
		Password:      tokenRes.AccessToken,
		ServerAddress: defaultRegistry,
	}, nil
}

// GetAll returns a map of all credentials in the backing store. If the backing
// store contains credentials for the official registry, these are refreshed/processed
// according to the same rules as Get.
func (o *oauthStore) GetAll() (map[string]types.AuthConfig, error) {
	// fetch all authconfigs from backing store
	allAuths, err := o.backingStore.GetAll()
	if err != nil {
		return nil, err
	}

	// if there are no oauth-type credentials for the default registry,
	// we can return as-is
	tokenRes := o.fetchFromBackingStore()
	if tokenRes == nil {
		return allAuths, nil
	}

	// if there is an oauth-type entry, then we need to parse it/refresh it
	auth, err := o.Get(defaultRegistry)
	if err != nil {
		return nil, err
	}
	allAuths[defaultRegistry] = auth

	// delete access/refresh-token specific entries since the caller
	// doesn't care about those
	delete(allAuths, accessTokenServerAddress)
	delete(allAuths, refreshTokenServerAddress)

	return allAuths, err
}

// Erase removes the credentials from the backing store.
// If the address pertains to the default registry, and there are oauth-type
// credentials stored, it also revokes the refresh token with the tenant.
func (o *oauthStore) Erase(serverAddress string) error {
	if serverAddress != defaultRegistry {
		return o.backingStore.Erase(serverAddress)
	}

	refreshTokenAuth, err := o.backingStore.Get(refreshTokenServerAddress)
	if err == nil && refreshTokenAuth.Password != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = o.manager.Logout(ctx, refreshTokenAuth.Password)
		if err != nil {
			// todo(laurazard): actual message here
			fmt.Fprint(os.Stderr, "Failed to revoke refresh token with tenant.. Credentials will still be erased.")
		}
	}

	_ = o.backingStore.Erase(defaultRegistry)
	_ = o.backingStore.Erase(accessTokenServerAddress)
	_ = o.backingStore.Erase(refreshTokenServerAddress)
	return nil
}

// Store stores the provided credentials in the backing store.
// If the provided credentials represent oauth-type credentials for the default
// registry, then those are stored as separate entries in the backing store.
// If there are basic auths and we're storing an oauth login, the basic auth
// entry is removed from the backing store, and vice versa.
func (o *oauthStore) Store(auth types.AuthConfig) error {
	if auth.ServerAddress != defaultRegistry {
		return o.backingStore.Store(auth)
	}

	accessToken, refreshToken, err := oauth.SplitTokens(auth.Password)
	if err != nil {
		// not storing an oauth-type login, so just store the auth as-is
		return errors.Join(
			// first, remove oauth logins if we had any
			o.backingStore.Erase(accessTokenServerAddress),
			o.backingStore.Erase(refreshTokenServerAddress),
			o.backingStore.Store(auth),
		)
	}

	// erase basic auths before storing our oauth-type login
	_ = o.backingStore.Erase(defaultRegistry)
	return errors.Join(
		o.backingStore.Store(types.AuthConfig{
			Username:      auth.Username,
			Password:      accessToken,
			ServerAddress: accessTokenServerAddress,
		}),
		o.backingStore.Store(types.AuthConfig{
			Username:      auth.Username,
			Password:      refreshToken,
			ServerAddress: refreshTokenServerAddress,
		}),
	)
}

const (
	defaultRegistryHostname   = "index.docker.io/v1"
	accessTokenServerAddress  = "https://access-token." + defaultRegistryHostname
	refreshTokenServerAddress = "https://refresh-token." + defaultRegistryHostname
)

func (o *oauthStore) storeInBackingStore(tokenRes oauth.TokenResult) error {
	return errors.Join(
		o.backingStore.Store(types.AuthConfig{
			Username:      tokenRes.Claims.Domain.Username,
			Password:      tokenRes.AccessToken,
			ServerAddress: accessTokenServerAddress,
		}),
		o.backingStore.Store(types.AuthConfig{
			Username:      tokenRes.Claims.Domain.Username,
			Password:      tokenRes.RefreshToken,
			ServerAddress: refreshTokenServerAddress,
		}),
	)
}

func (o *oauthStore) fetchFromBackingStore() *oauth.TokenResult {
	accessTokenAuth, err := o.backingStore.Get(accessTokenServerAddress)
	if err != nil {
		return nil
	}
	refreshTokenAuth, err := o.backingStore.Get(refreshTokenServerAddress)
	if err != nil {
		return nil
	}
	claims, err := oauth.GetClaims(accessTokenAuth.Password)
	if err != nil {
		return nil
	}
	return &oauth.TokenResult{
		AccessToken:  accessTokenAuth.Password,
		RefreshToken: refreshTokenAuth.Password,
		Claims:       claims,
	}
}
