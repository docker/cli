package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestGetDeviceCode(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var clientID, audience, scope, path string
		expectedState := State{
			DeviceCode:      "aDeviceCode",
			UserCode:        "aUserCode",
			VerificationURI: "aVerificationURI",
			ExpiresIn:       60,
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			clientID = r.FormValue("client_id")
			audience = r.FormValue("audience")
			scope = r.FormValue("scope")
			path = r.URL.Path

			jsonState, err := json.Marshal(expectedState)
			assert.NilError(t, err)

			_, _ = w.Write(jsonState)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		state, err := api.GetDeviceCode(context.Background(), "anAudience")
		assert.NilError(t, err)

		assert.DeepEqual(t, expectedState, state)
		assert.Equal(t, clientID, "aClientID")
		assert.Equal(t, audience, "anAudience")
		assert.Equal(t, scope, "bork meow")
		assert.Equal(t, path, "/oauth/device/code")
	})

	t.Run("error w/ description", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			jsonState, err := json.Marshal(TokenResponse{
				ErrorDescription: "invalid audience",
			})
			assert.NilError(t, err)

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(jsonState)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		_, err := api.GetDeviceCode(context.Background(), "bad_audience")

		assert.ErrorContains(t, err, "invalid audience")
	})

	t.Run("general error", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "an error", http.StatusInternalServerError)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		_, err := api.GetDeviceCode(context.Background(), "anAudience")

		assert.ErrorContains(t, err, "unexpected response from tenant: 500 Internal Server Error")
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(2 * time.Second)
			http.Error(w, "an error", http.StatusInternalServerError)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(1 * time.Second)
			cancel()
		}()
		_, err := api.GetDeviceCode(ctx, "anAudience")

		assert.ErrorContains(t, err, "context canceled")
	})
}

func TestWaitForDeviceToken(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		expectedToken := TokenResponse{
			AccessToken:  "a-real-token",
			IDToken:      "",
			RefreshToken: "the-refresh-token",
			Scope:        "",
			ExpiresIn:    3600,
			TokenType:    "",
		}
		var respond atomic.Bool
		go func() {
			time.Sleep(5 * time.Second)
			respond.Store(true)
		}()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/token", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("grant_type"), "urn:ietf:params:oauth:grant-type:device_code")
			assert.Equal(t, r.FormValue("device_code"), "aDeviceCode")

			if respond.Load() {
				jsonState, err := json.Marshal(expectedToken)
				assert.NilError(t, err)
				w.Write(jsonState)
			} else {
				pendingError := "authorization_pending"
				jsonResponse, err := json.Marshal(TokenResponse{
					Error: &pendingError,
				})
				assert.NilError(t, err)
				w.Write(jsonResponse)
			}
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}
		state := State{
			DeviceCode: "aDeviceCode",
			UserCode:   "aUserCode",
			Interval:   1,
			ExpiresIn:  30,
		}
		token, err := api.WaitForDeviceToken(context.Background(), state)
		assert.NilError(t, err)

		assert.DeepEqual(t, token, expectedToken)
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/token", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("grant_type"), "urn:ietf:params:oauth:grant-type:device_code")
			assert.Equal(t, r.FormValue("device_code"), "aDeviceCode")

			pendingError := "authorization_pending"
			jsonResponse, err := json.Marshal(TokenResponse{
				Error: &pendingError,
			})
			assert.NilError(t, err)
			w.Write(jsonResponse)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}
		state := State{
			DeviceCode: "aDeviceCode",
			UserCode:   "aUserCode",
			Interval:   5,
			ExpiresIn:  1,
		}

		_, err := api.WaitForDeviceToken(context.Background(), state)

		assert.ErrorIs(t, err, ErrTimeout)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			pendingError := "authorization_pending"
			jsonResponse, err := json.Marshal(TokenResponse{
				Error: &pendingError,
			})
			assert.NilError(t, err)
			w.Write(jsonResponse)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}
		state := State{
			DeviceCode: "aDeviceCode",
			UserCode:   "aUserCode",
			Interval:   1,
			ExpiresIn:  5,
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(1 * time.Second)
			cancel()
		}()
		_, err := api.WaitForDeviceToken(ctx, state)

		assert.ErrorContains(t, err, "context canceled")
	})
}

func TestRevoke(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/revoke", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("token"), "v1.a-refresh-token")

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		err := api.RevokeToken(context.Background(), "v1.a-refresh-token")
		assert.NilError(t, err)
	})

	t.Run("unexpected response", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/revoke", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("token"), "v1.a-refresh-token")

			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		err := api.RevokeToken(context.Background(), "v1.a-refresh-token")
		assert.ErrorContains(t, err, "unexpected response from tenant: 404 Not Found")
	})

	t.Run("error w/ description", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonState, err := json.Marshal(TokenResponse{
				ErrorDescription: "invalid client id",
			})
			assert.NilError(t, err)

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(jsonState)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		err := api.RevokeToken(context.Background(), "v1.a-refresh-token")
		assert.ErrorContains(t, err, "invalid client id")
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/revoke", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("token"), "v1.a-refresh-token")

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := api.RevokeToken(ctx, "v1.a-refresh-token")

		assert.ErrorContains(t, err, "context canceled")
	})
}

func TestGetAutoPAT(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v2/access-tokens/desktop-generate", r.URL.Path)
			assert.Equal(t, "Bearer bork", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			marshalledResponse, err := json.Marshal(patGenerateResponse{
				Data: struct {
					Token string `json:"token"`
				}{
					Token: "a-docker-pat",
				},
			})
			assert.NilError(t, err)
			w.WriteHeader(http.StatusCreated)
			w.Write(marshalledResponse)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		pat, err := api.GetAutoPAT(context.Background(), ts.URL, TokenResponse{
			AccessToken: "bork",
		})
		assert.NilError(t, err)

		assert.Equal(t, "a-docker-pat", pat)
	})

	t.Run("general error", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}

		_, err := api.GetAutoPAT(context.Background(), ts.URL, TokenResponse{
			AccessToken: "bork",
		})
		assert.ErrorContains(t, err, "unexpected response from Hub: 500 Internal Server Error")
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v2/access-tokens/desktop-generate", r.URL.Path)
			assert.Equal(t, "Bearer bork", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			marshalledResponse, err := json.Marshal(patGenerateResponse{
				Data: struct {
					Token string `json:"token"`
				}{
					Token: "a-docker-pat",
				},
			})
			assert.NilError(t, err)

			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusCreated)
			w.Write(marshalledResponse)
		}))
		defer ts.Close()
		api := API{
			TenantURL: ts.URL,
			ClientID:  "aClientID",
			Scopes:    []string{"bork", "meow"},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pat, err := api.GetAutoPAT(ctx, ts.URL, TokenResponse{
			AccessToken: "bork",
		})

		assert.ErrorContains(t, err, "context canceled")
		assert.Equal(t, "", pat)
	})
}
