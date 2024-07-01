package manager

import (
	"context"
	"os"
	"testing"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/internal/oauth/api"
	"github.com/docker/cli/cli/oauth"
	"github.com/go-jose/go-jose/v3/jwt"
	"gotest.tools/v3/assert"
)

const (
	//nolint:lll
	validToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhYa3BCdDNyV3MyRy11YjlscEpncSJ9.eyJodHRwczovL2h1Yi5kb2NrZXIuY29tIjp7ImVtYWlsIjoiYm9ya0Bkb2NrZXIuY29tIiwic2Vzc2lvbl9pZCI6ImEtc2Vzc2lvbi1pZCIsInNvdXJjZSI6InNhbWxwIiwidXNlcm5hbWUiOiJib3JrISIsInV1aWQiOiIwMTIzLTQ1Njc4OSJ9LCJpc3MiOiJodHRwczovL2xvZ2luLmRvY2tlci5jb20vIiwic3ViIjoic2FtbHB8c2FtbHAtZG9ja2VyfGJvcmtAZG9ja2VyLmNvbSIsImF1ZCI6WyJodHRwczovL2F1ZGllbmNlLmNvbSJdLCJpYXQiOjE3MTk1MDI5MzksImV4cCI6MTcxOTUwNjUzOSwic2NvcGUiOiJvcGVuaWQgb2ZmbGluZV9hY2Nlc3MifQ.VUSp-9_SOvMPWJPRrSh7p4kSPoye4DA3kyd2I0TW0QtxYSRq7xCzNj0NC_ywlPlKBFBeXKm4mh93d1vBSh79I9Heq5tj0Fr4KH77U5xJRMEpjHqoT5jxMEU1hYXX92xctnagBMXxDvzUfu3Yf0tvYSA0RRoGbGTHfdYYRwOrGbwQ75Qg1dyIxUkwsG053eYX2XkmLGxymEMgIq_gWksgAamOc40_0OCdGr-MmDeD2HyGUa309aGltzQUw7Z0zG1AKSXy3WwfMHdWNFioTAvQphwEyY3US8ybSJi78upSFTjwUcryMeHUwQ3uV9PxwPMyPoYxo1izVB-OUJxM8RqEbg"
	//nolint:lll
	newerToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhYa3BCdDNyV3MyRy11YjlscEpncSJ9.eyJodHRwczovL2h1Yi5kb2NrZXIuY29tIjp7ImVtYWlsIjoiYm9ya0Bkb2NrZXIuY29tIiwic2Vzc2lvbl9pZCI6ImEtc2Vzc2lvbi1pZCIsInNvdXJjZSI6InNhbWxwIiwidXNlcm5hbWUiOiJib3JrISIsInV1aWQiOiIwMTIzLTQ1Njc4OSJ9LCJpc3MiOiJodHRwczovL2xvZ2luLmRvY2tlci5jb20vIiwic3ViIjoic2FtbHB8c2FtbHAtZG9ja2VyfGJvcmtAZG9ja2VyLmNvbSIsImF1ZCI6WyJodHRwczovL2F1ZGllbmNlLmNvbSJdLCJpYXQiOjI3MTk1MDI5MzksImV4cCI6MjcxOTUwNjUzOSwic2NvcGUiOiJvcGVuaWQgb2ZmbGluZV9hY2Nlc3MifQ.VUSp-9_SOvMPWJPRrSh7p4kSPoye4DA3kyd2I0TW0QtxYSRq7xCzNj0NC_ywlPlKBFBeXKm4mh93d1vBSh79I9Heq5tj0Fr4KH77U5xJRMEpjHqoT5jxMEU1hYXX92xctnagBMXxDvzUfu3Yf0tvYSA0RRoGbGTHfdYYRwOrGbwQ75Qg1dyIxUkwsG053eYX2XkmLGxymEMgIq_gWksgAamOc40_0OCdGr-MmDeD2HyGUa309aGltzQUw7Z0zG1AKSXy3WwfMHdWNFioTAvQphwEyY3US8ybSJi78upSFTjwUcryMeHUwQ3uV9PxwPMyPoYxo1izVB-OUJxM8RqEbg"
)

var (
	expiry           = jwt.NumericDate(1719506539)
	issuedAt         = jwt.NumericDate(1719502939)
	validParsedToken = oauth.TokenResult{
		AccessToken:  validToken,
		RefreshToken: "refresh-token",
		Claims: oauth.Claims{
			Claims: jwt.Claims{
				Issuer:  "https://login.docker.com/",
				Subject: "samlp|samlp-docker|bork@docker.com",
				Audience: jwt.Audience{
					"https://audience.com",
				},
				Expiry:   &expiry,
				IssuedAt: &issuedAt,
			},
			Domain: oauth.DomainClaims{
				UUID:      "0123-456789",
				Email:     "bork@docker.com",
				Username:  "bork!",
				Source:    "samlp",
				SessionID: "a-session-id",
			},
			Scope: "openid offline_access",
		},
	}
)

func TestLoginDevice(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		credStore := &testCredStore{
			creds: make(map[string]types.AuthConfig),
		}
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
		api := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
		}
		manager := OAuthManager{
			audience:  "https://hub.docker.com",
			api:       api,
			credStore: credStore,
			openBrowser: func(url string) error {
				return nil
			},
		}

		res, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.NilError(t, err)

		assert.Equal(t, receivedAudience, "https://hub.docker.com")
		assert.Equal(t, receivedState, expectedState)
		assert.DeepEqual(t, res, validParsedToken)
	})

	t.Run("timeout", func(t *testing.T) {
		credStore := &testCredStore{
			creds: make(map[string]types.AuthConfig),
		}
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
			api:       a,
			credStore: credStore,
			openBrowser: func(url string) error {
				return nil
			},
		}

		_, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.ErrorContains(t, err, "login failed: timed out waiting for device token")
	})

	t.Run("stores in cred store", func(t *testing.T) {
		credStore := &testCredStore{
			creds: make(map[string]types.AuthConfig),
		}
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
		a := &testAPI{
			getDeviceToken:     getDeviceToken,
			waitForDeviceToken: waitForDeviceToken,
		}
		manager := OAuthManager{
			api:       a,
			credStore: credStore,
			openBrowser: func(url string) error {
				return nil
			},
		}

		res, err := manager.LoginDevice(context.Background(), os.Stderr)
		assert.NilError(t, err)

		assert.Equal(t, res.AccessToken, validToken)
		assert.Equal(t, credStore.creds[registryAuthKey+accessTokenKey].Password, validToken)
		assert.Equal(t, credStore.creds[registryAuthKey+refreshTokenKey].Password, "refresh-token")
	})
}

func TestLogout(t *testing.T) {
	credStore := &testCredStore{
		creds: map[string]types.AuthConfig{
			registryAuthKey + accessTokenKey: {
				ServerAddress: registryAuthKey + accessTokenKey,
				Username:      "username",
				Password:      "password",
			},
			registryAuthKey + refreshTokenKey: {
				ServerAddress: registryAuthKey + refreshTokenKey,
				Username:      "username",
				Password:      "password",
			},
		},
	}
	a := &testAPI{
		logoutURL: "test-logout-url",
	}
	var browserOpenURL string
	manager := OAuthManager{
		api:       a,
		credStore: credStore,
		openBrowser: func(url string) error {
			browserOpenURL = url
			return nil
		},
	}

	err := manager.Logout()
	assert.NilError(t, err)

	assert.Equal(t, browserOpenURL, "test-logout-url")
	_, ok := credStore.creds[registryAuthKey+accessTokenKey]
	assert.Check(t, !ok)
	_, ok = credStore.creds[registryAuthKey+refreshTokenKey]
	assert.Check(t, !ok)
}

func TestRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		credStore := &testCredStore{
			creds: map[string]types.AuthConfig{
				registryAuthKey + accessTokenKey: {
					ServerAddress: registryAuthKey + accessTokenKey,
					Username:      "username",
					Password:      validToken,
				},
				registryAuthKey + refreshTokenKey: {
					ServerAddress: registryAuthKey + refreshTokenKey,
					Username:      "username",
					Password:      "old-refresh-token",
				},
			},
		}
		var receivedRefreshToken string
		a := &testAPI{
			refresh: func(token string) (api.TokenResponse, error) {
				receivedRefreshToken = token
				return api.TokenResponse{
					AccessToken:  newerToken,
					RefreshToken: "new-refresh-token",
				}, nil
			},
		}
		manager := OAuthManager{
			api:       a,
			credStore: credStore,
		}

		res, err := manager.RefreshToken()
		assert.NilError(t, err)

		assert.Equal(t, receivedRefreshToken, "old-refresh-token")
		assert.Equal(t, credStore.creds[registryAuthKey+refreshTokenKey].Password, "new-refresh-token")
		assert.Equal(t, res.AccessToken, newerToken)
	})

	t.Run("no token", func(t *testing.T) {
		credStore := &testCredStore{
			creds: map[string]types.AuthConfig{},
		}
		a := &testAPI{
			refresh: func(token string) (api.TokenResponse, error) {
				return api.TokenResponse{
					AccessToken:  validToken,
					RefreshToken: "new-refresh-token",
				}, nil
			},
		}
		manager := OAuthManager{
			api:       a,
			credStore: credStore,
		}

		_, err := manager.RefreshToken()
		assert.ErrorIs(t, err, ErrNoCreds)
	})
}

var _ api.OAuthAPI = &testAPI{}

type testAPI struct {
	logoutURL          string
	getDeviceToken     func(audience string) (api.State, error)
	waitForDeviceToken func(state api.State) (api.TokenResponse, error)
	refresh            func(token string) (api.TokenResponse, error)
}

func (t *testAPI) GetDeviceCode(audience string) (api.State, error) {
	if t.getDeviceToken != nil {
		return t.getDeviceToken(audience)
	}
	return api.State{}, nil
}

func (t *testAPI) WaitForDeviceToken(state api.State) (api.TokenResponse, error) {
	if t.waitForDeviceToken != nil {
		return t.waitForDeviceToken(state)
	}
	return api.TokenResponse{}, nil
}

func (t *testAPI) Refresh(token string) (api.TokenResponse, error) {
	if t.refresh != nil {
		return t.refresh(token)
	}
	return api.TokenResponse{}, nil
}

func (t *testAPI) LogoutURL() string {
	return t.logoutURL
}

var _ credentials.Store = &testCredStore{}

type testCredStore struct {
	creds map[string]types.AuthConfig
}

func (t *testCredStore) Get(serverAddress string) (types.AuthConfig, error) {
	return t.creds[serverAddress], nil
}

func (t *testCredStore) GetAll() (map[string]types.AuthConfig, error) {
	return t.creds, nil
}

func (t *testCredStore) Store(authConfig types.AuthConfig) error {
	t.creds[authConfig.ServerAddress] = authConfig
	return nil
}

func (t *testCredStore) Erase(serverAddress string) error {
	delete(t.creds, serverAddress)
	return nil
}
