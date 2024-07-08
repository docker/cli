package manager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/cli/cli/config/credentials/internal/oauth/api"
	"github.com/docker/cli/cli/config/credentials/internal/oauth/util"
	"github.com/docker/cli/cli/oauth"
)

// OAuthManager is the manager
type OAuthManager struct {
	api         api.OAuthAPI
	audience    string
	tenant      string
	openBrowser func(string) error
}

// OAuthManagerOptions is the options used for New to create a new auth manager.
type OAuthManagerOptions struct {
	Audience    string
	ClientID    string
	Scopes      []string
	Tenant      string
	DeviceName  string
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
		openBrowser: openBrowser,
	}, nil
}

// LoginDevice launches the device authentication flow with the tenant, printing instructions
// to the provided writer and attempting to open the browser for the user to authenticate.
// Once complete, the retrieved tokens are stored and returned.
func (m *OAuthManager) LoginDevice(ctx context.Context, w io.Writer) (res oauth.TokenResult, err error) {
	state, err := m.api.GetDeviceCode(ctx, m.audience)
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
		tokenRes, err := m.api.WaitForDeviceToken(ctx, state)
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

	claims, err := oauth.GetClaims(tokenRes.AccessToken)
	if err != nil {
		return res, fmt.Errorf("login failed: %w", err)
	}

	res.Tenant = m.tenant
	res.AccessToken = tokenRes.AccessToken
	res.RefreshToken = tokenRes.RefreshToken
	res.Claims = claims

	return res, nil
}

// Logout logs out of the session for the client and removes tokens from the storage provider.
func (m *OAuthManager) Logout(ctx context.Context) error {
	return errors.Join(
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

// RefreshToken fetches credentials from the store, refreshes them, stores the new tokens
// and returns them.
// If there are no credentials in the store, ErrNoCreds is returned.
func (m OAuthManager) RefreshToken(ctx context.Context, refreshToken string) (res oauth.TokenResult, err error) {
	refreshRes, err := m.api.Refresh(ctx, refreshToken)
	if err != nil {
		return res, err
	}

	// todo(laurazard)
	// select {
	// case <-ctx.Done():
	// 	return "", ctx.Err()
	// default:
	// }

	claims, err := oauth.GetClaims(refreshRes.AccessToken)
	if err != nil {
		return res, err
	}

	res.Tenant = m.tenant
	res.AccessToken = refreshRes.AccessToken
	res.RefreshToken = refreshRes.RefreshToken
	res.Claims = claims
	return res, nil
}
