package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/cli/cli/internal/oauth/util"
)

type OAuthAPI interface {
	GetDeviceCode(audience string) (State, error)
	WaitForDeviceToken(state State) (TokenResponse, error)
	Refresh(token string) (TokenResponse, error)
	LogoutURL() string
}

// API represents API interactions with Auth0.
type API struct {
	// BaseURL is the base used for each request to Auth0.
	BaseURL string
	// ClientID is the client ID for the application to auth with the tenant.
	ClientID string
	// Scopes are the scopes that are requested during the device auth flow.
	Scopes []string
	// Client is the client that is used for calls.
	Client util.Client
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

// GetDeviceCode returns device code authorization information from Auth0.
func (a API) GetDeviceCode(audience string) (state State, err error) {
	data := url.Values{
		"client_id": {a.ClientID},
		"audience":  {audience},
		"scope":     {strings.Join(a.Scopes, " ")},
	}

	deviceCodeURL := a.BaseURL + "/oauth/device/code"
	resp, err := a.Client.PostForm(deviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var body map[string]any
		err = json.NewDecoder(resp.Body).Decode(&body)
		if errorDescription, ok := body["error_description"].(string); ok {
			return state, errors.New(errorDescription)
		}
		return state, fmt.Errorf("failed to get device code: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&state)

	return
}

// WaitForDeviceToken polls to get tokens based on the device code set up. This
// only works in a device auth flow.
func (a API) WaitForDeviceToken(state State) (TokenResponse, error) {
	ticker := time.NewTicker(state.IntervalDuration())
	timeout := time.After(time.Duration(state.ExpiresIn) * time.Second)

	for {
		select {
		case <-ticker.C:
			res, err := a.getDeviceToken(state)
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
			ticker.Stop()
			return TokenResponse{}, ErrTimeout
		}
	}
}

// getToken calls the token endpoint of Auth0 and returns the response.
func (a API) getDeviceToken(state State) (res TokenResponse, err error) {
	data := url.Values{
		"client_id":   {a.ClientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {state.DeviceCode},
	}
	oauthTokenURL := a.BaseURL + "/oauth/token"

	resp, err := a.Client.PostForm(oauthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return res, fmt.Errorf("failed to get code: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	_ = resp.Body.Close()

	return
}

// Refresh returns new tokens based on the refresh token.
func (a API) Refresh(token string) (res TokenResponse, err error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {a.ClientID},
		"refresh_token": {token},
	}

	refreshURL := a.BaseURL + "/oauth/token"
	//nolint:gosec // Ignore G107: Potential HTTP request made with variable url
	resp, err := http.PostForm(refreshURL, data)
	if err != nil {
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	_ = resp.Body.Close()

	return
}

func (a API) LogoutURL() string {
	return fmt.Sprintf("%s/v2/logout?client_id=%s", a.BaseURL, a.ClientID)
}
