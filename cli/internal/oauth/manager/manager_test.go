package manager

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth/api"
	"gotest.tools/v3/assert"
)

const (
	//nolint:lll
	validToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhYa3BCdDNyV3MyRy11YjlscEpncSJ9.eyJodHRwczovL2h1Yi5kb2NrZXIuY29tIjp7ImVtYWlsIjoiYm9ya0Bkb2NrZXIuY29tIiwic2Vzc2lvbl9pZCI6ImEtc2Vzc2lvbi1pZCIsInNvdXJjZSI6InNhbWxwIiwidXNlcm5hbWUiOiJib3JrISIsInV1aWQiOiIwMTIzLTQ1Njc4OSJ9LCJpc3MiOiJodHRwczovL2xvZ2luLmRvY2tlci5jb20vIiwic3ViIjoic2FtbHB8c2FtbHAtZG9ja2VyfGJvcmtAZG9ja2VyLmNvbSIsImF1ZCI6WyJodHRwczovL2F1ZGllbmNlLmNvbSJdLCJpYXQiOjE3MTk1MDI5MzksImV4cCI6MTcxOTUwNjUzOSwic2NvcGUiOiJvcGVuaWQgb2ZmbGluZV9hY2Nlc3MifQ.VUSp-9_SOvMPWJPRrSh7p4kSPoye4DA3kyd2I0TW0QtxYSRq7xCzNj0NC_ywlPlKBFBeXKm4mh93d1vBSh79I9Heq5tj0Fr4KH77U5xJRMEpjHqoT5jxMEU1hYXX92xctnagBMXxDvzUfu3Yf0tvYSA0RRoGbGTHfdYYRwOrGbwQ75Qg1dyIxUkwsG053eYX2XkmLGxymEMgIq_gWksgAamOc40_0OCdGr-MmDeD2HyGUa309aGltzQUw7Z0zG1AKSXy3WwfMHdWNFioTAvQphwEyY3US8ybSJi78upSFTjwUcryMeHUwQ3uV9PxwPMyPoYxo1izVB-OUJxM8RqEbg"
)

// parsed token:
// {
// 	"https://hub.docker.com": {
// 	  "email": "bork@docker.com",
// 	  "session_id": "a-session-id",
// 	  "source": "samlp",
// 	  "username": "bork!",
// 	  "uuid": "0123-456789"
// 	},
// 	"iss": "https://login.docker.com/",
// 	"sub": "samlp|samlp-docker|bork@docker.com",
// 	"aud": [
// 	  "https://audience.com"
// 	],
// 	"iat": 1719502939,
// 	"exp": 1719506539,
// 	"scope": "openid offline_access"
//   }

func TestLoginDevice(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		expectedState := api.State{
			DeviceCode:      "device-code",
			UserCode:        "0123-4567",
			VerificationURI: "an-url",
			ExpiresIn:       300,
		}
		var receivedAudience string
		getDeviceToken := func(audience string) (api.State, error) {
			receivedAudience = audience
			return expectedState, nil
		}
		var receivedState api.State
		waitForDeviceToken := func(state api.State) (api.TokenResponse, error) {
			receivedState = state
			return api.TokenResponse{
				AccessToken:  validToken,
				RefreshToken: "refresh-token",
			}, nil
		}
		var receivedAccessToken, getPatReceivedAudience string
		getAutoPat := func(audience string, res api.TokenResponse) (string, error) {
			receivedAccessToken = res.AccessToken
			getPatReceivedAudience = audience
			return "a-pat", nil
		}
		api := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
			getAutoPAT:         getAutoPat,
		}
		store := newStore(map[string]types.AuthConfig{})
		manager := OAuthManager{
			store:    credentials.NewFileStore(store),
			audience: "https://hub.docker.com",
			api:      api,
			openBrowser: func(url string) error {
				return nil
			},
		}

		authConfig, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.NilError(t, err)

		assert.Equal(t, receivedAudience, "https://hub.docker.com")
		assert.Equal(t, receivedState, expectedState)
		assert.DeepEqual(t, authConfig, &types.AuthConfig{
			Username:      "bork!",
			Password:      "a-pat",
			ServerAddress: "https://index.docker.io/v1/",
		})
		assert.Equal(t, receivedAccessToken, validToken)
		assert.Equal(t, getPatReceivedAudience, "https://hub.docker.com")
	})

	t.Run("stores in cred store", func(t *testing.T) {
		getDeviceToken := func(audience string) (api.State, error) {
			return api.State{
				DeviceCode: "device-code",
				UserCode:   "0123-4567",
			}, nil
		}
		waitForDeviceToken := func(state api.State) (api.TokenResponse, error) {
			return api.TokenResponse{
				AccessToken:  validToken,
				RefreshToken: "refresh-token",
			}, nil
		}
		getAutoPAT := func(audience string, res api.TokenResponse) (string, error) {
			return "a-pat", nil
		}
		a := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
			getAutoPAT:         getAutoPAT,
		}
		store := newStore(map[string]types.AuthConfig{})
		manager := OAuthManager{
			clientID: "client-id",
			store:    credentials.NewFileStore(store),
			api:      a,
			openBrowser: func(url string) error {
				return nil
			},
		}

		authConfig, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.NilError(t, err)

		assert.Equal(t, authConfig.Password, "a-pat")
		assert.Equal(t, authConfig.Username, "bork!")

		assert.Equal(t, len(store.configs), 2)
		assert.Equal(t, store.configs["https://index.docker.io/v1/access-token"].Password, validToken)
		assert.Equal(t, store.configs["https://index.docker.io/v1/refresh-token"].Password, "refresh-token..client-id")
	})

	t.Run("timeout", func(t *testing.T) {
		getDeviceToken := func(audience string) (api.State, error) {
			return api.State{
				DeviceCode:      "device-code",
				UserCode:        "0123-4567",
				VerificationURI: "an-url",
				ExpiresIn:       300,
			}, nil
		}
		waitForDeviceToken := func(state api.State) (api.TokenResponse, error) {
			return api.TokenResponse{}, api.ErrTimeout
		}
		a := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
		}
		manager := OAuthManager{
			api: a,
			openBrowser: func(url string) error {
				return nil
			},
		}

		_, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.ErrorContains(t, err, "failed waiting for authentication: timed out waiting for device token")
	})

	t.Run("canceled context", func(t *testing.T) {
		getDeviceToken := func(audience string) (api.State, error) {
			return api.State{
				DeviceCode: "device-code",
				UserCode:   "0123-4567",
			}, nil
		}
		waitForDeviceToken := func(state api.State) (api.TokenResponse, error) {
			// make sure that the context is cancelled before this returns
			time.Sleep(500 * time.Millisecond)
			return api.TokenResponse{
				AccessToken:  validToken,
				RefreshToken: "refresh-token",
			}, nil
		}
		a := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
		}
		manager := OAuthManager{
			api: a,
			openBrowser: func(url string) error {
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := manager.LoginDevice(ctx, os.Stderr)
		assert.ErrorContains(t, err, "login canceled")
	})
}

func TestLogout(t *testing.T) {
	t.Run("successfully revokes token", func(t *testing.T) {
		var receivedToken string
		a := &testAPI{
			revokeToken: func(token string) error {
				receivedToken = token
				return nil
			},
		}
		store := newStore(map[string]types.AuthConfig{
			"https://index.docker.io/v1/access-token": {
				Password: validToken,
			},
			"https://index.docker.io/v1/refresh-token": {
				Password: "a-refresh-token..client-id",
			},
		})
		manager := OAuthManager{
			store: credentials.NewFileStore(store),
			api:   a,
		}

		err := manager.Logout(context.Background())
		assert.NilError(t, err)

		assert.Equal(t, receivedToken, "a-refresh-token")
		assert.Equal(t, len(store.configs), 0)
	})

	t.Run("error revoking token", func(t *testing.T) {
		a := &testAPI{
			revokeToken: func(token string) error {
				return errors.New("couldn't reach tenant")
			},
		}
		store := newStore(map[string]types.AuthConfig{
			"https://index.docker.io/v1/access-token": {
				Password: validToken,
			},
			"https://index.docker.io/v1/refresh-token": {
				Password: "a-refresh-token..client-id",
			},
		})
		manager := OAuthManager{
			store: credentials.NewFileStore(store),
			api:   a,
		}

		err := manager.Logout(context.Background())
		assert.ErrorContains(t, err, "credentials erased successfully, but there was a failure to revoke the OAuth refresh token with the tenant: couldn't reach tenant")

		assert.Equal(t, len(store.configs), 0)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		var triedRevoke bool
		a := &testAPI{
			revokeToken: func(token string) error {
				triedRevoke = true
				return nil
			},
		}
		store := newStore(map[string]types.AuthConfig{
			"https://index.docker.io/v1/access-token": {
				Password: validToken,
			},
			"https://index.docker.io/v1/refresh-token": {
				Password: "a-refresh-token-without-client-id",
			},
		})
		manager := OAuthManager{
			store: credentials.NewFileStore(store),
			api:   a,
		}

		err := manager.Logout(context.Background())
		assert.NilError(t, err)

		assert.Check(t, !triedRevoke)
	})

	t.Run("no refresh token", func(t *testing.T) {
		a := &testAPI{}
		var triedRevoke bool
		revokeToken := func(token string) error {
			triedRevoke = true
			return nil
		}
		a.revokeToken = revokeToken
		store := newStore(map[string]types.AuthConfig{})
		manager := OAuthManager{
			store: credentials.NewFileStore(store),
			api:   a,
		}

		err := manager.Logout(context.Background())
		assert.NilError(t, err)

		assert.Check(t, !triedRevoke)
	})
}

var _ api.OAuthAPI = &testAPI{}

type testAPI struct {
	getDeviceToken     func(audience string) (api.State, error)
	waitForDeviceToken func(state api.State) (api.TokenResponse, error)
	refresh            func(token string) (api.TokenResponse, error)
	revokeToken        func(token string) error
	getAutoPAT         func(audience string, res api.TokenResponse) (string, error)
}

func (t *testAPI) GetDeviceCode(_ context.Context, audience string) (api.State, error) {
	if t.getDeviceToken != nil {
		return t.getDeviceToken(audience)
	}
	return api.State{}, nil
}

func (t *testAPI) WaitForDeviceToken(_ context.Context, state api.State) (api.TokenResponse, error) {
	if t.waitForDeviceToken != nil {
		return t.waitForDeviceToken(state)
	}
	return api.TokenResponse{}, nil
}

func (t *testAPI) Refresh(_ context.Context, token string) (api.TokenResponse, error) {
	if t.refresh != nil {
		return t.refresh(token)
	}
	return api.TokenResponse{}, nil
}

func (t *testAPI) RevokeToken(_ context.Context, token string) error {
	if t.revokeToken != nil {
		return t.revokeToken(token)
	}
	return nil
}

func (t *testAPI) GetAutoPAT(_ context.Context, audience string, res api.TokenResponse) (string, error) {
	if t.getAutoPAT != nil {
		return t.getAutoPAT(audience, res)
	}
	return "", nil
}

type fakeStore struct {
	configs map[string]types.AuthConfig
}

func (f *fakeStore) Save() error {
	return nil
}

func (f *fakeStore) GetAuthConfigs() map[string]types.AuthConfig {
	return f.configs
}

func (f *fakeStore) GetFilename() string {
	return "/tmp/docker-fakestore"
}

func newStore(auths map[string]types.AuthConfig) *fakeStore {
	return &fakeStore{configs: auths}
}
