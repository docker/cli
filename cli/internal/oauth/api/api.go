// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.21

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli/version"
)

type OAuthAPI interface {
	GetDeviceCode(ctx context.Context, audience string) (State, error)
	WaitForDeviceToken(ctx context.Context, state State) (TokenResponse, error)
	RevokeToken(ctx context.Context, refreshToken string) error
	GetAutoPAT(ctx context.Context, audience string, res TokenResponse) (string, error)
}

// API represents API interactions with Auth0.
type API struct {
	// TenantURL is the base used for each request to Auth0.
	TenantURL string
	// ClientID is the client ID for the application to auth with the tenant.
	ClientID string
	// Scopes are the scopes that are requested during the device auth flow.
	Scopes []string
}

// TokenResponse represents the response of the /oauth/token route.
type TokenResponse struct {
	AccessToken      string  `json:"access_token"`
	IDToken          string  `json:"id_token"`
	RefreshToken     string  `json:"refresh_token"`
	Scope            string  `json:"scope"`
	ExpiresIn        int     `json:"expires_in"`
	TokenType        string  `json:"token_type"`
	Error            *string `json:"error,omitempty"`
	ErrorDescription string  `json:"error_description,omitempty"`
}

var ErrTimeout = errors.New("timed out waiting for device token")

// GetDeviceCode initiates the device-code auth flow with the tenant.
// The state returned contains the device code that the user must use to
// authenticate, as well as the URL to visit, etc.
func (a API) GetDeviceCode(ctx context.Context, audience string) (State, error) {
	data := url.Values{
		"client_id": {a.ClientID},
		"audience":  {audience},
		"scope":     {strings.Join(a.Scopes, " ")},
	}

	deviceCodeURL := a.TenantURL + "/oauth/device/code"
	resp, err := postForm(ctx, deviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return State{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return State{}, tryDecodeOAuthError(resp)
	}

	var state State
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		return state, fmt.Errorf("failed to get device code: %w", err)
	}

	return state, nil
}

func tryDecodeOAuthError(resp *http.Response) error {
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		if errorDescription, ok := body["error_description"].(string); ok {
			return errors.New(errorDescription)
		}
	}
	return errors.New("unexpected response from tenant: " + resp.Status)
}

// WaitForDeviceToken polls the tenant to get access/refresh tokens for the user.
// This should be called after GetDeviceCode, and will block until the user has
// authenticated or we have reached the time limit for authenticating (based on
// the response from GetDeviceCode).
func (a API) WaitForDeviceToken(ctx context.Context, state State) (TokenResponse, error) {
	ticker := time.NewTicker(state.IntervalDuration())
	defer ticker.Stop()
	timeout := time.After(state.ExpiryDuration())

	for {
		select {
		case <-ctx.Done():
			return TokenResponse{}, ctx.Err()
		case <-ticker.C:
			res, err := a.getDeviceToken(ctx, state)
			if err != nil {
				return res, err
			}

			if res.Error != nil {
				if *res.Error == "authorization_pending" {
					continue
				}

				return res, errors.New(res.ErrorDescription)
			}

			return res, nil
		case <-timeout:
			return TokenResponse{}, ErrTimeout
		}
	}
}

// getToken calls the token endpoint of Auth0 and returns the response.
func (a API) getDeviceToken(ctx context.Context, state State) (TokenResponse, error) {
	data := url.Values{
		"client_id":   {a.ClientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {state.DeviceCode},
	}
	oauthTokenURL := a.TenantURL + "/oauth/token"

	resp, err := postForm(ctx, oauthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("failed to get tokens: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// this endpoint returns a 403 with an `authorization_pending` error until the
	// user has authenticated, so we don't check the status code here and instead
	// decode the response and check for the error.
	var res TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return res, fmt.Errorf("failed to decode response: %w", err)
	}

	return res, nil
}

// RevokeToken revokes a refresh token with the tenant so that it can no longer
// be used to get new tokens.
func (a API) RevokeToken(ctx context.Context, refreshToken string) error {
	data := url.Values{
		"client_id": {a.ClientID},
		"token":     {refreshToken},
	}

	revokeURL := a.TenantURL + "/oauth/revoke"
	resp, err := postForm(ctx, revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return tryDecodeOAuthError(resp)
	}

	return nil
}

func postForm(ctx context.Context, reqURL string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, data)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cliVersion := strings.ReplaceAll(version.Version, ".", "_")
	req.Header.Set("User-Agent", fmt.Sprintf("docker-cli:%s:%s-%s", cliVersion, runtime.GOOS, runtime.GOARCH))

	return http.DefaultClient.Do(req)
}

func (a API) GetAutoPAT(ctx context.Context, audience string, res TokenResponse) (string, error) {
	patURL := audience + "/v2/access-tokens/desktop-generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, patURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+res.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected response from Hub: %s", resp.Status)
	}

	var response patGenerateResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Data.Token, nil
}

type patGenerateResponse struct {
	Data struct {
		Token string `json:"token"`
	}
}
