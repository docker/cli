package manager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth"
	"github.com/docker/cli/cli/internal/oauth/api"
	"github.com/docker/docker/registry"
	"github.com/morikuni/aec"
	"github.com/sirupsen/logrus"

	"github.com/pkg/browser"
)

// OAuthManager is the manager responsible for handling authentication
// flows with the oauth tenant.
type OAuthManager struct {
	store       credentials.Store
	tenant      string
	audience    string
	clientID    string
	api         api.OAuthAPI
	openBrowser func(string) error
}

// OAuthManagerOptions are the options used for New to create a new auth manager.
type OAuthManagerOptions struct {
	Store       credentials.Store
	Audience    string
	ClientID    string
	Scopes      []string
	Tenant      string
	DeviceName  string
	OpenBrowser func(string) error
}

func New(options OAuthManagerOptions) *OAuthManager {
	scopes := []string{"openid", "offline_access"}
	if len(options.Scopes) > 0 {
		scopes = options.Scopes
	}

	openBrowser := options.OpenBrowser
	if openBrowser == nil {
		// Prevent errors from missing binaries (like xdg-open) from
		// cluttering the output. We can handle errors ourselves.
		browser.Stdout = io.Discard
		browser.Stderr = io.Discard
		openBrowser = browser.OpenURL
	}

	return &OAuthManager{
		clientID: options.ClientID,
		audience: options.Audience,
		tenant:   options.Tenant,
		store:    options.Store,
		api: api.API{
			TenantURL: "https://" + options.Tenant,
			ClientID:  options.ClientID,
			Scopes:    scopes,
		},
		openBrowser: openBrowser,
	}
}

var ErrDeviceLoginStartFail = errors.New("failed to start device code flow login")

// LoginDevice launches the device authentication flow with the tenant,
// printing instructions to the provided writer and attempting to open the
// browser for the user to authenticate.
// After the user completes the browser login, LoginDevice uses the retrieved
// tokens to create a Hub PAT which is returned to the caller.
// The retrieved tokens are stored in the credentials store (under a separate
// key), and the refresh token is concatenated with the client ID.
func (m *OAuthManager) LoginDevice(ctx context.Context, w io.Writer) (*types.AuthConfig, error) {
	state, err := m.api.GetDeviceCode(ctx, m.audience)
	if err != nil {
		logrus.Debugf("failed to start device code login: %v", err)
		return nil, ErrDeviceLoginStartFail
	}

	if state.UserCode == "" {
		logrus.Debugf("failed to start device code login: missing user code")
		return nil, ErrDeviceLoginStartFail
	}

	_, _ = fmt.Fprintln(w, aec.Bold.Apply("\nUSING WEB-BASED LOGIN"))
	_, _ = fmt.Fprintln(w, "To sign in with credentials on the command line, use 'docker login -u <username>'")
	_, _ = fmt.Fprintf(w, "\nYour one-time device confirmation code is: "+aec.Bold.Apply("%s\n"), state.UserCode)
	_, _ = fmt.Fprintf(w, aec.Bold.Apply("Press ENTER")+" to open your browser or submit your device code here: "+aec.Underline.Apply("%s\n"), strings.Split(state.VerificationURI, "?")[0])

	tokenResChan := make(chan api.TokenResponse)
	waitForTokenErrChan := make(chan error)
	go func() {
		tokenRes, err := m.api.WaitForDeviceToken(ctx, state)
		if err != nil {
			waitForTokenErrChan <- err
			return
		}
		tokenResChan <- tokenRes
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
		_ = m.openBrowser(state.VerificationURI)
	}()

	_, _ = fmt.Fprint(w, "\nWaiting for authentication in the browser…\n")
	var tokenRes api.TokenResponse
	select {
	case <-ctx.Done():
		return nil, errors.New("login canceled")
	case err := <-waitForTokenErrChan:
		return nil, fmt.Errorf("failed waiting for authentication: %w", err)
	case tokenRes = <-tokenResChan:
	}

	claims, err := oauth.GetClaims(tokenRes.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	err = m.storeTokensInStore(tokenRes, claims.Domain.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to store tokens: %w", err)
	}

	pat, err := m.api.GetAutoPAT(ctx, m.audience, tokenRes)
	if err != nil {
		return nil, err
	}

	return &types.AuthConfig{
		Username:      claims.Domain.Username,
		Password:      pat,
		ServerAddress: registry.IndexServer,
	}, nil
}

// Logout fetches the refresh token from the store and revokes it
// with the configured oauth tenant. The stored access and refresh
// tokens are then erased from the store.
// If the refresh token is not found in the store, an error is not
// returned.
func (m *OAuthManager) Logout(ctx context.Context) error {
	refreshConfig, err := m.store.Get(refreshTokenKey)
	if err != nil {
		return err
	}
	if refreshConfig.Password == "" {
		return nil
	}
	parts := strings.Split(refreshConfig.Password, "..")
	if len(parts) != 2 {
		// the token wasn't stored by the CLI, so don't revoke it
		// or erase it from the store/error
		return nil
	}
	// erase the token from the store first, that way
	// if the revoke fails, the user can try to logout again
	if err := m.eraseTokensFromStore(); err != nil {
		return fmt.Errorf("failed to erase tokens: %w", err)
	}
	if err := m.api.RevokeToken(ctx, parts[0]); err != nil {
		return fmt.Errorf("credentials erased successfully, but there was a failure to revoke the OAuth refresh token with the tenant: %w", err)
	}
	return nil
}

const (
	accessTokenKey  = registry.IndexServer + "access-token"
	refreshTokenKey = registry.IndexServer + "refresh-token"
)

func (m *OAuthManager) storeTokensInStore(tokens api.TokenResponse, username string) error {
	return errors.Join(
		m.store.Store(types.AuthConfig{
			Username:      username,
			Password:      tokens.AccessToken,
			ServerAddress: accessTokenKey,
		}),
		m.store.Store(types.AuthConfig{
			Username:      username,
			Password:      tokens.RefreshToken + ".." + m.clientID,
			ServerAddress: refreshTokenKey,
		}),
	)
}

func (m *OAuthManager) eraseTokensFromStore() error {
	return errors.Join(
		m.store.Erase(accessTokenKey),
		m.store.Erase(refreshTokenKey),
	)
}
