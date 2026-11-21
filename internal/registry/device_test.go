package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestDiscoverDeviceLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer realm="http://token.example/token",service="registry",device_authorization_endpoint="http://token.example/device"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)

	login, err := DiscoverDeviceLogin(context.Background(), APIEndpoint{URL: u}, "test")
	assert.NilError(t, err)
	assert.Check(t, is.Equal("http://token.example/device", login.DeviceEndpoint))
	assert.Check(t, is.Equal("http://token.example/token", login.TokenEndpoint))
}

func TestDiscoverDeviceLoginUsesTokenEndpointParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="registry",device_authorization_endpoint="http://token.example/device",token_endpoint="http://token.example/oauth/token"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)

	login, err := DiscoverDeviceLogin(context.Background(), APIEndpoint{URL: u}, "")
	assert.NilError(t, err)
	assert.Check(t, is.Equal("http://token.example/oauth/token", login.TokenEndpoint))
}

func TestDiscoverDeviceLoginAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)

	_, err = DiscoverDeviceLogin(context.Background(), APIEndpoint{URL: u}, "")
	assert.Check(t, is.ErrorIs(err, ErrNoDeviceLogin))
}

func TestDiscoverDeviceLoginRejectsHTTPEndpointsOnHTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="http://token.example/token",device_authorization_endpoint="http://token.example/device"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)

	_, err = DiscoverDeviceLogin(context.Background(), APIEndpoint{URL: u, TLSConfig: srv.Client().Transport.(*http.Transport).TLSClientConfig}, "")
	assert.Check(t, is.ErrorIs(err, ErrDeviceLoginStartFail))
}

func TestDeviceCodeExchange(t *testing.T) {
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil || r.PostForm.Get("client_id") != AuthClientID {
			http.Error(w, "client", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_code":"dev","user_code":"ABCD-EFGH","verification_uri":"http://example.test/activate","verification_uri_complete":"http://example.test/activate?user_code=ABCD-EFGH","expires_in":60,"interval":5}`))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := polls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
			return
		}
		_, _ = w.Write([]byte(`{"username":"alice","secret":"s3cret"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	code, err := RequestDeviceCode(context.Background(), srv.Client(), srv.URL+"/device")
	assert.NilError(t, err)
	assert.Check(t, is.Equal("ABCD-EFGH", code.UserCode))
	assert.Check(t, is.Equal(5*time.Second, code.Interval))

	cred, err := PollDeviceToken(context.Background(), srv.Client(), srv.URL+"/token", code.DeviceCode, code.Interval, func(context.Context, time.Duration) error {
		return nil
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("alice", cred.Username))
	assert.Check(t, is.Equal("s3cret", cred.Secret))
	assert.Check(t, is.Equal(int32(2), polls.Load()))
}

func TestPollDeviceTokenDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"access_denied","error_description":"the sign-in was refused"}`))
	}))
	t.Cleanup(srv.Close)

	_, err := PollDeviceToken(context.Background(), srv.Client(), srv.URL, "dev", 5*time.Second, func(context.Context, time.Duration) error {
		return nil
	})
	assert.ErrorContains(t, err, "the sign-in was refused")
}
