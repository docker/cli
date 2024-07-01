package manager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth/api"
	"github.com/docker/cli/cli/internal/oauth/util"
	"github.com/docker/cli/cli/oauth"
)

// OAuthManager is the manager
type OAuthManager struct {
	api         api.OAuthAPI
	audience    string
	tenant      string
	credStore   credentials.Store
	openBrowser func(string) error
}

// OAuthManagerOptions is the options used for New to create a new auth manager.
type OAuthManagerOptions struct {
	Audience    string
	ClientID    string
	Scopes      []string
	Tenant      string
	DeviceName  string
	Store       credentials.Store
	OpenBrowser func(string) error
}

func New(options OAuthManagerOptions) (*OAuthManager, error) {
	scopes := []string{"openid", "offline_access"}
	if len(options.Scopes) > 0 {
		scopes = options.Scopes
	}

	openBrowser := util.OpenBrowser
	if options.OpenBrowser != nil {
		openBrowser = options.OpenBrowser
	}

	return &OAuthManager{
		audience: options.Audience,
		api: api.API{
			BaseURL:  "https://" + options.Tenant,
			ClientID: options.ClientID,
			Scopes:   scopes,
			Client: util.Client{
				UserAgent: options.DeviceName,
			},
		},
		tenant:      options.Tenant,
		credStore:   options.Store,
		openBrowser: openBrowser,
	}, nil
}

// LoginDevice launches the device authentication flow with the tenant, printing instructions
// to the provided writer and attempting to open the browser for the user to authenticate.
// Once complete, the retrieved tokens are stored and returned.
func (m *OAuthManager) LoginDevice(ctx context.Context, w io.Writer) (res oauth.TokenResult, err error) {
	state, err := m.api.GetDeviceCode(m.audience)
	if err != nil {
		return res, fmt.Errorf("login failed: %w", err)
	}

	if state.UserCode == "" {
		return res, errors.New("login failed: no user code returned")
	}

	_, _ = fmt.Fprintln(w, "\nYou will be signed in using a web-based login.")
	_, _ = fmt.Fprintln(w, "To sign in with credentials on the command line, use 'docker login -u <username>'")
	_, _ = fmt.Fprintf(w, "\nYour one-time device confirmation code is: %s\n", state.UserCode)
	_, _ = fmt.Fprint(w, "\nPress ENTER to open the browser.\n")
	_, _ = fmt.Fprintf(w, "Or open the URL manually: %s.\n", strings.Split(state.VerificationURI, "?")[0])

	tokenResChan := make(chan api.TokenResponse)
	waitForTokenErrChan := make(chan error)
	go func() {
		tokenRes, err := m.api.WaitForDeviceToken(state)
		if err != nil {
			waitForTokenErrChan <- err
			return
		}
		tokenResChan <- tokenRes
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		_ = m.openBrowser(state.VerificationURI)
	}()

	_, _ = fmt.Fprint(w, "\nWaiting for authentication in the browser...\n")
	var tokenRes api.TokenResponse
	select {
	case <-ctx.Done():
		return res, errors.New("login canceled")
	case err := <-waitForTokenErrChan:
		return res, fmt.Errorf("login failed: %w", err)
	case tokenRes = <-tokenResChan:
	}

	claims, err := util.GetClaims(tokenRes.AccessToken)
	if err != nil {
		return res, fmt.Errorf("login failed: %w", err)
	}

	res.Tenant = m.tenant
	res.AccessToken = tokenRes.AccessToken
	res.RefreshToken = tokenRes.RefreshToken
	res.Claims = claims

	if err = m.storeTokensInStore(tokenRes.AccessToken, tokenRes.RefreshToken); err != nil {
		return res, fmt.Errorf("login failed: %w", err)
	}
	return res, nil
}

// Logout logs out of the session for the client and removes tokens from the storage provider.
func (m *OAuthManager) Logout() error {
	return errors.Join(
		m.eraseTokensFromStore(),
		m.openBrowser(m.api.LogoutURL()),
	)
}

var (
	// ErrNoCreds is returned by RefreshToken when the store does not contain credentials
	// for the official registry.
	ErrNoCreds = errors.New("no credentials found")

	// ErrUnexpiredToken is returned by RefreshToken when the token is not expired.
	ErrUnexpiredToken = errors.New("token is not expired")
)

const minimumTokenLifetime = 20 * time.Minute

// RefreshToken fetches credentials from the store, refreshes them, stores the new tokens
// and returns them.
// If there are no credentials in the store, ErrNoCreds is returned.
func (m OAuthManager) RefreshToken() (res oauth.TokenResult, err error) {
	access, refresh, err := m.fetchTokensFromStore()
	if err != nil {
		return res, err
	}
	if access == "" {
		return res, ErrNoCreds
	}

	claims, err := util.GetClaims(access)
	if err != nil {
		return res, err
	}
	if claims.Expiry.Time().After(time.Now().Add(minimumTokenLifetime)) {
		return res, ErrUnexpiredToken
	}

	refreshRes, err := m.api.Refresh(refresh)
	if err != nil {
		return res, err
	}

	err = m.storeTokensInStore(refreshRes.AccessToken, refreshRes.RefreshToken)
	if err != nil {
		return res, err
	}

	claims, err = util.GetClaims(refreshRes.AccessToken)
	if err != nil {
		return res, err
	}

	res.Tenant = m.tenant
	res.AccessToken = refreshRes.AccessToken
	res.RefreshToken = refreshRes.RefreshToken
	res.Claims = claims
	return res, nil
}

const (
	registryAuthKey = "https://index.docker.io/v1/"
	accessTokenKey  = "access-token"
	refreshTokenKey = "refresh-token"
)

func (m *OAuthManager) fetchTokensFromStore() (access, refresh string, err error) {
	accessAuth, err := m.credStore.Get(registryAuthKey + accessTokenKey)
	if err != nil {
		return access, refresh, fmt.Errorf("failed to fetch access token: %w", err)
	}
	access = accessAuth.Password

	refreshAuth, err := m.credStore.Get(registryAuthKey + refreshTokenKey)
	if err != nil {
		return access, refresh, fmt.Errorf("failed to fetch refresh token: %w", err)
	}
	refresh = refreshAuth.Password

	return
}

func (m *OAuthManager) storeTokensInStore(accessToken, refreshToken string) error {
	claims, err := util.GetClaims(accessToken)
	if err != nil {
		return err
	}
	return errors.Join(
		m.credStore.Store(types.AuthConfig{
			ServerAddress: registryAuthKey + accessTokenKey,
			Username:      claims.Domain.Username,
			Password:      accessToken,
		}), m.credStore.Store(types.AuthConfig{
			ServerAddress: registryAuthKey + refreshTokenKey,
			Username:      claims.Domain.Username,
			Password:      refreshToken,
		}))
}

func (m *OAuthManager) eraseTokensFromStore() error {
	return errors.Join(
		m.credStore.Erase(registryAuthKey+accessTokenKey),
		m.credStore.Erase(registryAuthKey+refreshTokenKey),
	)
}
