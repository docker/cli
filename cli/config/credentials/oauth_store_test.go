package credentials

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/oauth"
	"github.com/docker/docker/registry"
	"github.com/go-jose/go-jose/v3/jwt"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const (
	//nolint:lll
	validExpiredToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhYa3BCdDNyV3MyRy11YjlscEpncSJ9.eyJodHRwczovL2h1Yi5kb2NrZXIuY29tIjp7ImVtYWlsIjoiYm9ya0Bkb2NrZXIuY29tIiwic2Vzc2lvbl9pZCI6ImEtc2Vzc2lvbi1pZCIsInNvdXJjZSI6InNhbWxwIiwidXNlcm5hbWUiOiJib3JrISIsInV1aWQiOiIwMTIzLTQ1Njc4OSJ9LCJpc3MiOiJodHRwczovL2xvZ2luLmRvY2tlci5jb20vIiwic3ViIjoic2FtbHB8c2FtbHAtZG9ja2VyfGJvcmtAZG9ja2VyLmNvbSIsImF1ZCI6WyJodHRwczovL2F1ZGllbmNlLmNvbSJdLCJpYXQiOjE3MTk1MDI5MzksImV4cCI6MTcxOTUwNjUzOSwic2NvcGUiOiJvcGVuaWQgb2ZmbGluZV9hY2Nlc3MifQ.VUSp-9_SOvMPWJPRrSh7p4kSPoye4DA3kyd2I0TW0QtxYSRq7xCzNj0NC_ywlPlKBFBeXKm4mh93d1vBSh79I9Heq5tj0Fr4KH77U5xJRMEpjHqoT5jxMEU1hYXX92xctnagBMXxDvzUfu3Yf0tvYSA0RRoGbGTHfdYYRwOrGbwQ75Qg1dyIxUkwsG053eYX2XkmLGxymEMgIq_gWksgAamOc40_0OCdGr-MmDeD2HyGUa309aGltzQUw7Z0zG1AKSXy3WwfMHdWNFioTAvQphwEyY3US8ybSJi78upSFTjwUcryMeHUwQ3uV9PxwPMyPoYxo1izVB-OUJxM8RqEbg"
	//nolint:lll
	validNotExpiredToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhYa3BCdDNyV3MyRy11YjlscEpncSJ9.eyJodHRwczovL2h1Yi5kb2NrZXIuY29tIjp7ImVtYWlsIjoiYm9ya0Bkb2NrZXIuY29tIiwic2Vzc2lvbl9pZCI6ImEtc2Vzc2lvbi1pZCIsInNvdXJjZSI6InNhbWxwIiwidXNlcm5hbWUiOiJib3JrISIsInV1aWQiOiIwMTIzLTQ1Njc4OSJ9LCJpc3MiOiJodHRwczovL2xvZ2luLmRvY2tlci5jb20vIiwic3ViIjoic2FtbHB8c2FtbHAtZG9ja2VyfGJvcmtAZG9ja2VyLmNvbSIsImF1ZCI6WyJodHRwczovL2F1ZGllbmNlLmNvbSJdLCJpYXQiOjI3MTk1MDI5MzksImV4cCI6NDg3Mzc4MDQ2Niwic2NvcGUiOiJvcGVuaWQgb2ZmbGluZV9hY2Nlc3MifQ.VUSp-9_SOvMPWJPRrSh7p4kSPoye4DA3kyd2I0TW0QtxYSRq7xCzNj0NC_ywlPlKBFBeXKm4mh93d1vBSh79I9Heq5tj0Fr4KH77U5xJRMEpjHqoT5jxMEU1hYXX92xctnagBMXxDvzUfu3Yf0tvYSA0RRoGbGTHfdYYRwOrGbwQ75Qg1dyIxUkwsG053eYX2XkmLGxymEMgIq_gWksgAamOc40_0OCdGr-MmDeD2HyGUa309aGltzQUw7Z0zG1AKSXy3WwfMHdWNFioTAvQphwEyY3US8ybSJi78upSFTjwUcryMeHUwQ3uV9PxwPMyPoYxo1izVB-OUJxM8RqEbg"
)

func TestOAuthStoreGet(t *testing.T) {
	t.Run("official registry", func(t *testing.T) {
		t.Run("valid credentials - no refresh", func(t *testing.T) {
			auths := map[string]types.AuthConfig{
				registry.IndexServer: {
					Username:      "bork!",
					Email:         "bork@docker.com",
					Password:      validNotExpiredToken + "..refresh-token",
					ServerAddress: registry.IndexServer,
				},
			}
			s := &oauthStore{
				backingStore: NewFileStore(newStore(auths)),
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Password:      validNotExpiredToken,
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
		})

		t.Run("no credentials - login", func(t *testing.T) {
			auths := map[string]types.AuthConfig{}
			f := newStore(auths)
			manager := &testManager{
				loginDevice: func() (oauth.TokenResult, error) {
					return oauth.TokenResult{
						AccessToken:  "abcd1234",
						RefreshToken: "efgh5678",
						Claims: oauth.Claims{
							Claims: jwt.Claims{
								Expiry: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
							},
							Domain: oauth.DomainClaims{Username: "bork!", Email: "bork@docker.com"},
						},
					}, nil
				},
			}
			s := &oauthStore{
				backingStore: NewFileStore(f),
				manager:      manager,
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
			assert.DeepEqual(t, auths[registry.IndexServer], types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234..efgh5678",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
		})

		t.Run("expired credentials - refresh", func(t *testing.T) {
			f := newStore(map[string]types.AuthConfig{
				registry.IndexServer: {
					Username:      "bork!",
					Email:         "bork@docker.com",
					Password:      validExpiredToken + "..refresh-token",
					ServerAddress: registry.IndexServer,
				},
			})
			var receivedRefreshToken string
			manager := &testManager{
				refresh: func(token string) (oauth.TokenResult, error) {
					receivedRefreshToken = token
					return oauth.TokenResult{
						AccessToken:  "abcd1234",
						RefreshToken: "efgh5678",
						Claims: oauth.Claims{
							Claims: jwt.Claims{
								Expiry: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
							},
							Domain: oauth.DomainClaims{Username: "bork!", Email: "bork@docker.com"},
						},
					}, nil
				},
			}
			s := &oauthStore{
				backingStore: NewFileStore(f),
				manager:      manager,
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
			assert.Equal(t, receivedRefreshToken, "refresh-token")
			assert.DeepEqual(t, f.GetAuthConfigs()[registry.IndexServer], types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234..efgh5678",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
		})

		t.Run("expired credentials - refresh fails - login", func(t *testing.T) {
			f := newStore(map[string]types.AuthConfig{
				registry.IndexServer: {
					Username:      "bork!",
					Email:         "bork@docker.com",
					Password:      validExpiredToken + "..refresh-token",
					ServerAddress: registry.IndexServer,
				},
			})
			var loginCalled bool
			manager := &testManager{
				refresh: func(_ string) (oauth.TokenResult, error) {
					return oauth.TokenResult{}, errors.New("program failed")
				},
				loginDevice: func() (oauth.TokenResult, error) {
					loginCalled = true
					return oauth.TokenResult{
						AccessToken:  "abcd1234",
						RefreshToken: "efgh5678",
						Claims: oauth.Claims{
							Claims: jwt.Claims{
								Expiry: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
							},
							Domain: oauth.DomainClaims{Username: "bork!", Email: "bork@docker.com"},
						},
					}, nil
				},
			}
			s := &oauthStore{
				backingStore: NewFileStore(f),
				manager:      manager,
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.Check(t, loginCalled)
			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
			assert.DeepEqual(t, f.GetAuthConfigs()[registry.IndexServer], types.AuthConfig{
				Username:      "bork!",
				Password:      "abcd1234..efgh5678",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			})
		})

		t.Run("old non-access token credentials", func(t *testing.T) {
			f := newStore(map[string]types.AuthConfig{
				registry.IndexServer: {
					Username:      "bork!",
					Email:         "bork@docker.com",
					Password:      "a-password",
					ServerAddress: registry.IndexServer,
				},
			})
			s := &oauthStore{
				backingStore: NewFileStore(f),
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Email:         "bork@docker.com",
				Password:      "a-password",
				ServerAddress: registry.IndexServer,
			})
		})

		t.Run("old non-access token credentials w/ ..", func(t *testing.T) {
			f := newStore(map[string]types.AuthConfig{
				registry.IndexServer: {
					Username:      "bork!",
					Email:         "bork@docker.com",
					Password:      "a-password..with-dots",
					ServerAddress: registry.IndexServer,
				},
			})
			s := &oauthStore{
				backingStore: NewFileStore(f),
			}

			auth, err := s.Get(registry.IndexServer)
			assert.NilError(t, err)

			assert.DeepEqual(t, auth, types.AuthConfig{
				Username:      "bork!",
				Email:         "bork@docker.com",
				Password:      "a-password..with-dots",
				ServerAddress: registry.IndexServer,
			})
		})
	})

	t.Run("defers when different registry", func(t *testing.T) {
		auth := types.AuthConfig{
			Username:      "foo",
			Password:      "bar",
			Email:         "foo@example.com",
			ServerAddress: validServerAddress,
		}
		f := newStore(map[string]types.AuthConfig{
			validServerAddress2: auth,
		})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}
		actual, err := s.Get(validServerAddress2)
		assert.NilError(t, err)

		assert.DeepEqual(t, actual, auth)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}

		as, err := s.GetAll()
		assert.NilError(t, err)

		assert.Check(t, is.Len(as, 0))
	})
	t.Run("1 - official registry", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{
			registry.IndexServer: {
				Username:      "bork!",
				Password:      validNotExpiredToken + "..refresh-token",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			},
		})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}

		as, err := s.GetAll()
		assert.NilError(t, err)

		assert.Equal(t, len(as), 1)
	})

	t.Run("1 - other registry", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{
			registry.IndexServer: {
				Username:      "bork!",
				Password:      "password",
				Email:         "bork@docker.com",
				ServerAddress: validServerAddress2,
			},
		})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}

		as, err := s.GetAll()
		assert.NilError(t, err)

		assert.Equal(t, len(as), 1)
	})

	t.Run("multiple - official and other registry", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{
			registry.IndexServer: {
				Username:      "bork!",
				Password:      validNotExpiredToken + "..refresh-token",
				Email:         "bork@docker.com",
				ServerAddress: registry.IndexServer,
			},
			validServerAddress2: {
				Username:      "foo",
				Password:      "bar",
				Email:         "bork@dockr.com",
				ServerAddress: validServerAddress2,
			},
		})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}

		as, err := s.GetAll()
		assert.NilError(t, err)

		assert.Equal(t, len(as), 2)
	})
}

func TestErase(t *testing.T) {
	t.Run("official registry", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{
			registry.IndexServer: {
				Email: "foo@example.com",
			},
		})
		var logoutCalled bool
		manager := &testManager{
			logout: func() error {
				logoutCalled = true
				return nil
			},
		}
		s := &oauthStore{
			backingStore: NewFileStore(f),
			manager:      manager,
		}
		err := s.Erase(registry.IndexServer)
		assert.NilError(t, err)

		assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
		assert.Check(t, logoutCalled)
	})

	t.Run("different registry", func(t *testing.T) {
		f := newStore(map[string]types.AuthConfig{
			validServerAddress2: {
				Email: "foo@example.com",
			},
		})
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}
		err := s.Erase(validServerAddress2)
		assert.NilError(t, err)
		assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
	})
}

func TestStore(t *testing.T) {
	t.Run("official registry", func(t *testing.T) {
		t.Run("regular credentials", func(t *testing.T) {
			f := newStore(make(map[string]types.AuthConfig))
			s := &oauthStore{
				backingStore: NewFileStore(f),
			}
			auth := types.AuthConfig{
				Username:      "foo",
				Password:      "bar",
				Email:         "foo@example.com",
				ServerAddress: registry.IndexServer,
			}
			err := s.Store(auth)
			assert.NilError(t, err)
			assert.Check(t, is.Len(f.GetAuthConfigs(), 1))
		})

		t.Run("access token", func(t *testing.T) {
			f := newStore(make(map[string]types.AuthConfig))
			s := &oauthStore{
				backingStore: NewFileStore(f),
			}
			auth := types.AuthConfig{
				Username:      "foo",
				Password:      validNotExpiredToken + "..refresh-token",
				Email:         "foo@example.com",
				ServerAddress: registry.IndexServer,
			}
			err := s.Store(auth)
			assert.NilError(t, err)
			assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
		})
	})

	t.Run("other registry", func(t *testing.T) {
		f := newStore(make(map[string]types.AuthConfig))
		s := &oauthStore{
			backingStore: NewFileStore(f),
		}
		auth := types.AuthConfig{
			Username:      "foo",
			Password:      "bar",
			Email:         "foo@example.com",
			ServerAddress: validServerAddress2,
		}
		err := s.Store(auth)
		assert.NilError(t, err)
		assert.Check(t, is.Len(f.GetAuthConfigs(), 1))

		actual, ok := f.GetAuthConfigs()[validServerAddress2]
		assert.Check(t, ok)
		assert.DeepEqual(t, actual, auth)
	})
}

type testManager struct {
	loginDevice func() (oauth.TokenResult, error)
	logout      func() error
	refresh     func(token string) (oauth.TokenResult, error)
}

func (m *testManager) LoginDevice(_ context.Context, _ io.Writer) (oauth.TokenResult, error) {
	return m.loginDevice()
}

func (m *testManager) Logout(_ context.Context) error {
	return m.logout()
}

func (m *testManager) RefreshToken(_ context.Context, token string) (oauth.TokenResult, error) {
	return m.refresh(token)
}
