package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestLoginV2BasicAuthUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "alice" || pass != "secret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)
	endpoint := APIEndpoint{URL: u}
	ctx := context.Background()

	_, err = loginV2(ctx, &registry.AuthConfig{Username: "alice", Password: "wrong"}, endpoint, "docker-test")
	assert.ErrorContains(t, err, "401")
	assert.Check(t, errdefs.IsUnauthorized(err))

	token, err := loginV2(ctx, &registry.AuthConfig{Username: "alice", Password: "secret"}, endpoint, "docker-test")
	assert.NilError(t, err)
	assert.Check(t, is.Equal("", token))
}
