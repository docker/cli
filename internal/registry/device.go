package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/distribution/registry/client/auth/challenge"
)

const (
	deviceAuthParam    = "device_authorization_endpoint"
	tokenEndpointParam = "token_endpoint"
	deviceClientID     = AuthClientID

	// deviceGrantType is the RFC 8628 device-code grant.
	deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"
)

// ErrNoDeviceLogin means the registry did not advertise a device-code login.
var ErrNoDeviceLogin = errors.New("registry did not advertise device login")

// ErrDeviceLoginStartFail means the device-code flow could not be started.
// Callers should fall back to a username and password prompt.
var ErrDeviceLoginStartFail = errors.New("failed to start device code flow login")

// DeviceLogin is a registry that offered a browser login on its /v2/ challenge.
type DeviceLogin struct {
	// DeviceEndpoint is where the client starts the device-code flow.
	DeviceEndpoint string
	// TokenEndpoint is where the client polls until the browser login finishes.
	TokenEndpoint string
}

// DeviceCode is the RFC 8628 device authorization response.
type DeviceCode struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// DeviceCredential is the registry credential returned when the poll succeeds.
type DeviceCredential struct {
	Username      string
	Secret        string
	IdentityToken string
}

// DiscoverDeviceLogin reads a device-code login offer from GET /v2/.
// ErrNoDeviceLogin is returned when the registry answers but does not advertise one.
func DiscoverDeviceLogin(ctx context.Context, endpoint APIEndpoint, userAgent string) (DeviceLogin, error) {
	ping, err := url.Parse(strings.TrimRight(endpoint.URL.String(), "/") + "/v2/")
	if err != nil {
		return DeviceLogin{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ping.String(), http.NoBody)
	if err != nil {
		return DeviceLogin{}, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	client := &http.Client{Transport: newTransport(endpoint.TLSConfig), Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return DeviceLogin{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusUnauthorized {
		return DeviceLogin{}, ErrNoDeviceLogin
	}
	for _, item := range challenge.ResponseChallenges(resp) {
		if item.Scheme != "bearer" {
			continue
		}
		deviceEndpoint := strings.TrimSpace(item.Parameters[deviceAuthParam])
		if deviceEndpoint == "" {
			continue
		}
		tokenEndpoint := strings.TrimSpace(item.Parameters[tokenEndpointParam])
		if tokenEndpoint == "" {
			tokenEndpoint = strings.TrimSpace(item.Parameters["realm"])
		}
		login := DeviceLogin{DeviceEndpoint: deviceEndpoint, TokenEndpoint: tokenEndpoint}
		if err := login.validate(endpoint.URL.Scheme); err != nil {
			return DeviceLogin{}, fmt.Errorf("%w: %w", ErrDeviceLoginStartFail, err)
		}
		return login, nil
	}
	return DeviceLogin{}, ErrNoDeviceLogin
}

func (d DeviceLogin) validate(registryScheme string) error {
	deviceURL, err := parseHTTPURL(d.DeviceEndpoint)
	if err != nil {
		return fmt.Errorf("device authorization endpoint: %w", err)
	}
	tokenURL, err := parseHTTPURL(d.TokenEndpoint)
	if err != nil {
		return fmt.Errorf("token endpoint: %w", err)
	}
	if registryScheme == "https" && (deviceURL.Scheme != "https" || tokenURL.Scheme != "https") {
		return errors.New("device login endpoints must use https")
	}
	return nil
}

func parseHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("must be an absolute http(s) URL")
	}
	return u, nil
}

// RequestDeviceCode starts an RFC 8628 device authorization against endpoint.
func RequestDeviceCode(ctx context.Context, client *http.Client, endpoint string) (DeviceCode, error) {
	form := url.Values{}
	form.Set("client_id", deviceClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return DeviceCode{}, fmt.Errorf("%w: %w", ErrDeviceLoginStartFail, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("%w: %w", ErrDeviceLoginStartFail, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return DeviceCode{}, fmt.Errorf("%w: %w", ErrDeviceLoginStartFail, err)
	}
	if resp.StatusCode != http.StatusOK {
		return DeviceCode{}, fmt.Errorf("%w: device authorization returned %s", ErrDeviceLoginStartFail, resp.Status)
	}
	var raw struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
		Error                   string `json:"error"`
		ErrorDescription        string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return DeviceCode{}, fmt.Errorf("%w: %w", ErrDeviceLoginStartFail, err)
	}
	if raw.Error != "" {
		return DeviceCode{}, fmt.Errorf("%w: %s", ErrDeviceLoginStartFail, deviceErrorText(raw.Error, raw.ErrorDescription))
	}
	if raw.DeviceCode == "" || raw.UserCode == "" || raw.VerificationURI == "" {
		return DeviceCode{}, fmt.Errorf("%w: device authorization response is incomplete", ErrDeviceLoginStartFail)
	}
	interval := time.Duration(raw.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	expires := time.Duration(raw.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = 5 * time.Minute
	}
	return DeviceCode{
		DeviceCode:              raw.DeviceCode,
		UserCode:                raw.UserCode,
		VerificationURI:         raw.VerificationURI,
		VerificationURIComplete: raw.VerificationURIComplete,
		ExpiresIn:               expires,
		Interval:                interval,
	}, nil
}

// PollDeviceToken polls the token endpoint until the browser login finishes,
// is refused, or ctx ends. sleep is used between polls so tests need not wait.
func PollDeviceToken(ctx context.Context, client *http.Client, tokenEndpoint, deviceCode string, interval time.Duration, sleep func(context.Context, time.Duration) error) (DeviceCredential, error) {
	if sleep == nil {
		sleep = sleepContext
	}
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return DeviceCredential{}, err
		}
		cred, pending, next, err := pollDeviceOnce(ctx, client, tokenEndpoint, deviceCode)
		if err != nil {
			return DeviceCredential{}, err
		}
		if !pending {
			return cred, nil
		}
		if next > 0 {
			interval += next
		}
		if err := sleep(ctx, interval); err != nil {
			return DeviceCredential{}, err
		}
	}
}

func pollDeviceOnce(ctx context.Context, client *http.Client, tokenEndpoint, deviceCode string) (DeviceCredential, bool, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", deviceGrantType)
	form.Set("device_code", deviceCode)
	form.Set("client_id", deviceClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return DeviceCredential{}, false, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return DeviceCredential{}, false, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return DeviceCredential{}, false, 0, err
	}
	var raw struct {
		Username         string `json:"username"`
		Secret           string `json:"secret"`
		IdentityToken    string `json:"identity_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return DeviceCredential{}, false, 0, fmt.Errorf("device token response: %w", err)
	}
	switch raw.Error {
	case "":
		if resp.StatusCode != http.StatusOK {
			return DeviceCredential{}, false, 0, fmt.Errorf("device token endpoint returned %s", resp.Status)
		}
		if raw.Username == "" || (raw.Secret == "" && raw.IdentityToken == "") {
			return DeviceCredential{}, false, 0, errors.New("device token response did not include a registry credential")
		}
		return DeviceCredential{Username: raw.Username, Secret: raw.Secret, IdentityToken: raw.IdentityToken}, false, 0, nil
	case "authorization_pending":
		return DeviceCredential{}, true, 0, nil
	case "slow_down":
		return DeviceCredential{}, true, 5 * time.Second, nil
	default:
		return DeviceCredential{}, false, 0, errors.New(deviceErrorText(raw.Error, raw.ErrorDescription))
	}
}

func deviceErrorText(code, description string) string {
	if description != "" {
		return description
	}
	return code
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
