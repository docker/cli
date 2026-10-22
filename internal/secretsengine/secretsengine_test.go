//go:build go1.26

package secretsengine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/registry"
	seclient "github.com/docker/secrets-engine/client"
	"github.com/docker/secrets-engine/client/dockerhub"
	"gotest.tools/v3/assert"
)

var testNow = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

type fakeHub struct {
	session      dockerhub.UserSession
	sessionErr   error
	profile      dockerhub.Profile
	profileErr   error
	sessionCalls atomic.Int32
}

func (f *fakeHub) GetDefaultSession(context.Context) (dockerhub.UserSession, error) {
	f.sessionCalls.Add(1)
	return f.session, f.sessionErr
}

func (f *fakeHub) GetDefaultProfile(context.Context) (dockerhub.Profile, error) {
	return f.profile, f.profileErr
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func hubSession(username string, expiresAt time.Time) dockerhub.UserSession {
	s := dockerhub.UserSession{AccessToken: "desktop-oauth-token"}
	s.Claims.Username = username
	if !expiresAt.IsZero() {
		s.Claims.ExpiresAt = &dockerhub.NumericDate{Time: expiresAt}
	}
	return s
}

type recordingFallback struct {
	mu    sync.Mutex
	hosts []string
}

func (f *recordingFallback) GetAuthConfig(registryHostname string) (types.AuthConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hosts = append(f.hosts, registryHostname)
	return cliCredentials(registryHostname), nil
}

func cliCredentials(registryHostname string) types.AuthConfig {
	return types.AuthConfig{Username: "cli-user", Password: "cli-password", ServerAddress: registryHostname}
}

func desktopCredentials() types.AuthConfig {
	return types.AuthConfig{Username: "hubuser", Password: "desktop-oauth-token", ServerAddress: registry.IndexServer}
}

func stagingCredentials() types.AuthConfig {
	return types.AuthConfig{Username: "stageuser", Password: "staging-oauth-token", ServerAddress: registry.StagingRegistryHost}
}

type testProvider struct {
	*Provider
	fallback *recordingFallback
	clock    *fakeClock
	staging  *fakeHub
}

func newTestProvider(hub HubClient) *testProvider {
	tp := &testProvider{
		fallback: &recordingFallback{},
		clock:    &fakeClock{t: testNow},
		staging:  &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile},
	}
	tp.Provider = New(tp.fallback, hub, tp.staging)
	for _, s := range []*session{tp.Provider.hub, tp.Provider.staging} {
		s.timeout = time.Second
		s.now = tp.clock.now
	}
	return tp
}

func (tp *testProvider) signInToStaging() {
	tp.staging.sessionErr = nil
	tp.staging.session = hubSession("stageuser", testNow.Add(time.Hour))
	tp.staging.session.AccessToken = "staging-oauth-token"
}

func connectError(reason seclient.ConnectReason) error {
	return &seclient.ConnectError{
		Reason:     reason,
		SocketPath: "/run/engine.sock",
		Err:        errors.New("dial unix /run/engine.sock"),
	}
}

func TestNewDesktopSharesOneSessionPerEnvironment(t *testing.T) {
	a := NewDesktop(&recordingFallback{})
	b := NewDesktop(&recordingFallback{})
	assert.Assert(t, a.hub == b.hub, "providers must share the process-wide Docker Hub session")
	assert.Assert(t, a.staging == b.staging, "providers must share the process-wide Docker Hub staging session")
	assert.Assert(t, a.hub != a.staging, "production and staging sessions must be separate")
}

func TestDockerHubUsesDesktopSession(t *testing.T) {
	for _, host := range []string{"docker.io", "index.docker.io", "registry-1.docker.io", registry.IndexServer} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub)

			got, err := p.GetAuthConfig(host)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, desktopCredentials())
			assert.Equal(t, got.IdentityToken, "", "an IdentityToken is sent as a refresh token")
			assert.Equal(t, len(p.fallback.hosts), 0)
		})
	}
}

func TestOtherRegistriesUseFallback(t *testing.T) {
	hosts := []string{
		"ghcr.io", "localhost:5000", "registry.example.com", "",
		registry.StagingRegistryHost + ".example.com", "example.com/" + registry.StagingRegistryHost,
		registry.DHIRegistryHost + ".example.com", "example.com/" + registry.DHIRegistryHost, "registry." + registry.DHIRegistryHost,
		"docker.io.example.com",
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub)
			p.signInToStaging()

			got, err := p.GetAuthConfig(host)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, cliCredentials(host))
			assert.DeepEqual(t, p.fallback.hosts, []string{host})
			assert.Equal(t, hub.sessionCalls.Load(), int32(0), "Docker Desktop must only be contacted for Docker Hub")
			assert.Equal(t, p.staging.sessionCalls.Load(), int32(0), "Docker Desktop must only be contacted for Docker Hub")
		})
	}
}

func TestDHIUsesDockerHubSession(t *testing.T) {
	for _, host := range []string{registry.DHIRegistryHost, "https://" + registry.DHIRegistryHost, "https://" + registry.DHIRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub)
			p.signInToStaging()

			want := desktopCredentials()
			want.ServerAddress = registry.DHIRegistryHost
			got, err := p.GetAuthConfig(host)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, want)
			assert.Equal(t, p.staging.sessionCalls.Load(), int32(0), "the staging session must not be read for DHI")

			got, err = p.GetAuthConfig("docker.io")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, desktopCredentials())
			assert.Equal(t, hub.sessionCalls.Load(), int32(1))
		})
	}
}

func TestStagingRegistryUsesStagingSession(t *testing.T) {
	for _, host := range []string{registry.StagingRegistryHost, "https://" + registry.StagingRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub)
			p.signInToStaging()

			got, err := p.GetAuthConfig(host)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, stagingCredentials())
			assert.Equal(t, hub.sessionCalls.Load(), int32(0), "the production session must not be read for the staging registry")
		})
	}
}

func TestEnvironmentsDoNotMix(t *testing.T) {
	t.Run("staging session is not used for docker.io", func(t *testing.T) {
		p := newTestProvider(&fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile})
		p.signInToStaging()

		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))

		got, err = p.GetAuthConfig(registry.StagingRegistryHost)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, stagingCredentials())
	})

	t.Run("production session is not used for the staging registry", func(t *testing.T) {
		p := newTestProvider(&fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))})

		got, err := p.GetAuthConfig(registry.StagingRegistryHost)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials(registry.StagingRegistryHost))

		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
	})
}

func TestUsernameFromProfile(t *testing.T) {
	hub := &fakeHub{
		session: hubSession("", testNow.Add(time.Hour)),
		profile: dockerhub.Profile{Username: "hubuser"},
	}
	p := newTestProvider(hub)

	got, err := p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
}

func TestFallsBackWithoutUsableSession(t *testing.T) {
	future := testNow.Add(time.Hour)
	tests := []struct {
		name string
		hub  *fakeHub
	}{
		{name: "secrets engine not running", hub: &fakeHub{sessionErr: connectError(seclient.ReasonNotRunning)}},
		{name: "secrets engine socket not accessible", hub: &fakeHub{sessionErr: connectError(seclient.ReasonPermissionDenied)}},
		{name: "nobody signed in", hub: &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile}},
		{name: "access denied", hub: &fakeHub{sessionErr: seclient.ErrAccessDenied}},
		{name: "no username", hub: &fakeHub{session: hubSession("", future)}},
		{name: "profile lookup fails", hub: &fakeHub{session: hubSession("", future), profileErr: errors.New("boom")}},
		{name: "expired", hub: &fakeHub{session: hubSession("hubuser", testNow.Add(-time.Minute))}},
		{name: "expires within the skew", hub: &fakeHub{session: hubSession("hubuser", testNow.Add(sessionExpirySkew/2))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newTestProvider(tc.hub)

			for range 2 {
				got, err := p.GetAuthConfig("docker.io")
				assert.NilError(t, err)
				assert.DeepEqual(t, got, cliCredentials("docker.io"))
			}
			assert.Equal(t, tc.hub.sessionCalls.Load(), int32(1), "Desktop must not be asked again within retryInterval")
		})
	}
}

func TestDesktopFallsBackWhenClientCannotBeCreated(t *testing.T) {
	var connects atomic.Int32
	s := newSession("Docker Hub", nil)
	s.connect = func() (HubClient, error) {
		connects.Add(1)
		return nil, errors.New("no socket path")
	}
	p := &Provider{fallback: &recordingFallback{}, hub: s}

	for range 2 {
		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))
	}
	assert.Equal(t, connects.Load(), int32(1))
}

func TestRetriesAfterInterval(t *testing.T) {
	hub := &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile}
	p := newTestProvider(hub)

	got, err := p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, cliCredentials("docker.io"))

	hub.sessionErr = nil
	hub.session = hubSession("hubuser", testNow.Add(time.Hour))

	p.clock.advance(retryInterval - time.Second)
	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, cliCredentials("docker.io"))
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	p.clock.advance(time.Second)
	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestReusesSessionUntilExpiry(t *testing.T) {
	expiresAt := testNow.Add(10 * time.Minute)
	hub := &fakeHub{session: hubSession("hubuser", expiresAt)}
	p := newTestProvider(hub)

	got, err := p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	p.clock.advance(5 * time.Minute)
	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	hub.session = hubSession("hubuser", expiresAt.Add(time.Hour))
	hub.session.AccessToken = "refreshed-token"
	p.clock.advance(5*time.Minute - sessionExpirySkew)

	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.Equal(t, got.Password, "refreshed-token")
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestRefetchesSessionWithoutExpiry(t *testing.T) {
	hub := &fakeHub{session: hubSession("hubuser", time.Time{})}
	p := newTestProvider(hub)

	got, err := p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	p.clock.advance(sessionTTL - time.Second)
	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	p.clock.advance(time.Second)
	got, err = p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestConcurrentLookupsShareOneFetch(t *testing.T) {
	hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
	p := newTestProvider(hub)

	const lookups = 20
	results := make([]types.AuthConfig, lookups)
	var wg sync.WaitGroup
	for i := range lookups {
		wg.Go(func() {
			auth, err := p.GetAuthConfig("docker.io")
			assert.Check(t, err)
			results[i] = auth
		})
	}
	wg.Wait()

	for _, auth := range results {
		assert.DeepEqual(t, auth, desktopCredentials())
	}
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))
}

func TestConnectSkipsMissingSocket(t *testing.T) {
	_, err := connect(filepath.Join(t.TempDir(), "engine.sock"))
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), err)
	assert.Assert(t, isSessionUnavailable(err))
}

func TestIsSessionUnavailable(t *testing.T) {
	for _, err := range []error{
		connectError(seclient.ReasonNotRunning),
		fmt.Errorf("get session: %w", connectError(seclient.ReasonNotRunning)),
		connectError(seclient.ReasonPermissionDenied),
		connectError(seclient.ReasonTimeout),
		connectError(seclient.ReasonUnknown),
		dockerhub.ErrNoSession,
		dockerhub.ErrNoDefaultProfile,
		errSessionExpired,
		fs.ErrNotExist,
	} {
		assert.Assert(t, isSessionUnavailable(err), err)
	}
	for _, err := range []error{
		seclient.ErrAccessDenied,
		errNoUsername,
		context.DeadlineExceeded,
	} {
		assert.Assert(t, !isSessionUnavailable(err), err)
	}
}
