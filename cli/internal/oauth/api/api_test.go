package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/cli/cli/internal/oauth/util"
	"gotest.tools/v3/assert"
)

func TestGetDeviceCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
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
			BaseURL:  ts.URL,
			ClientID: "aClientID",
			Scopes:   []string{"bork", "meow"},
			Client:   util.Client{},
		}

		state, err := api.GetDeviceCode("anAudience")
		assert.NilError(t, err)

		assert.DeepEqual(t, expectedState, state)
		assert.Equal(t, clientID, "aClientID")
		assert.Equal(t, audience, "anAudience")
		assert.Equal(t, scope, "bork meow")
		assert.Equal(t, path, "/oauth/device/code")
	})

	t.Run("error w/ description", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonState, err := json.Marshal(TokenResponse{
				ErrorDescription: "invalid audience",
			})
			assert.NilError(t, err)

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(jsonState)
		}))
		defer ts.Close()
		api := API{
			BaseURL:  ts.URL,
			ClientID: "aClientID",
			Scopes:   []string{"bork", "meow"},
			Client:   util.Client{},
		}

		_, err := api.GetDeviceCode("bad_audience")

		assert.ErrorContains(t, err, "invalid audience")
	})

	t.Run("general error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "an error", http.StatusInternalServerError)
		}))
		defer ts.Close()
		api := API{
			BaseURL:  ts.URL,
			ClientID: "aClientID",
			Scopes:   []string{"bork", "meow"},
			Client:   util.Client{},
		}

		_, err := api.GetDeviceCode("anAudience")

		assert.ErrorContains(t, err, "failed to get device code")
	})
}

func TestWaitForDeviceToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedToken := TokenResponse{
			AccessToken:  "a-real-token",
			IDToken:      "",
			RefreshToken: "the-refresh-token",
			Scope:        "",
			ExpiresIn:    3600,
			TokenType:    "",
		}
		var respond bool
		go func() {
			time.Sleep(10 * time.Second)
			respond = true
		}()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oauth/token", r.URL.Path)
			assert.Equal(t, r.FormValue("client_id"), "aClientID")
			assert.Equal(t, r.FormValue("grant_type"), "urn:ietf:params:oauth:grant-type:device_code")
			assert.Equal(t, r.FormValue("device_code"), "aDeviceCode")

			if respond {
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
			BaseURL:  ts.URL,
			ClientID: "aClientID",
			Scopes:   []string{"bork", "meow"},
			Client:   util.Client{},
		}
		state := State{
			DeviceCode: "aDeviceCode",
			UserCode:   "aUserCode",
			Interval:   1,
			ExpiresIn:  30,
		}
		token, err := api.WaitForDeviceToken(state)
		assert.NilError(t, err)

		assert.DeepEqual(t, token, expectedToken)
	})

	t.Run("timeout", func(t *testing.T) {
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
			BaseURL:  ts.URL,
			ClientID: "aClientID",
			Scopes:   []string{"bork", "meow"},
			Client:   util.Client{},
		}
		state := State{
			DeviceCode: "aDeviceCode",
			UserCode:   "aUserCode",
			Interval:   1,
			ExpiresIn:  1,
		}

		_, err := api.WaitForDeviceToken(state)

		assert.ErrorIs(t, err, ErrTimeout)
	})
}
