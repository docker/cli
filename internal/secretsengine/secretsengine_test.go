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

func desktopCredentials() types.AuthConfig {
	return types.AuthConfig{Username: "hubuser", Password: "desktop-oauth-token", ServerAddress: indexServer}
}

func stagingCredentials() types.AuthConfig {
	return types.AuthConfig{Username: "stageuser", Password: "staging-oauth-token", ServerAddress: stagingRegistryHost}
}

type testProvider struct {
	*provider
	clock    *fakeClock
	connects atomic.Int32
	staging  *fakeHub
}

func newTestProvider(hub hubSessions, connectErr error) *testProvider {
	tp := &testProvider{
		clock:   &fakeClock{t: testNow},
		staging: &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile},
	}
	tp.provider = &provider{
		hub: &session{
			name: "Docker Hub",
			connect: func() (hubSessions, error) {
				tp.connects.Add(1)
				if connectErr != nil {
					return nil, connectErr
				}
				return hub, nil
			},
			timeout: time.Second,
			now:     tp.clock.now,
		},
		staging: &session{
			name: "Docker Hub staging",
			connect: func() (hubSessions, error) {
				return tp.staging, nil
			},
			timeout: time.Second,
			now:     tp.clock.now,
		},
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

func TestDisableSkipsDockerDesktop(t *testing.T) {
	Disable()

	_, ok := AuthConfig("docker.io")
	assert.Assert(t, !ok)

	shared.hub.mu.Lock()
	defer shared.hub.mu.Unlock()
	assert.Assert(t, shared.hub.retryAt.IsZero(), "Docker Desktop must not be contacted once disabled")
	assert.Assert(t, shared.hub.hub == nil)
}

func TestSharedProviderHasOneSessionPerEnvironment(t *testing.T) {
	assert.Assert(t, shared.hub != nil)
	assert.Assert(t, shared.staging != nil)
	assert.Assert(t, shared.hub != shared.staging, "production and staging sessions must be separate")
}

func TestDockerHubUsesDesktopSession(t *testing.T) {
	for _, host := range []string{"docker.io", "index.docker.io", "registry-1.docker.io", indexServer} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub, nil)

			got, ok := p.authConfig(host)
			assert.Assert(t, ok)
			assert.DeepEqual(t, got, desktopCredentials())
			assert.Equal(t, got.IdentityToken, "", "an IdentityToken is sent as a refresh token")
		})
	}
}

func TestOtherRegistriesDoNotContactDesktop(t *testing.T) {
	hosts := []string{
		"ghcr.io", "localhost:5000", "registry.example.com", "",
		stagingRegistryHost + ".example.com", "example.com/" + stagingRegistryHost,
		dhiRegistryHost + ".example.com", "example.com/" + dhiRegistryHost, "registry." + dhiRegistryHost,
		"docker.io.example.com",
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub, nil)
			p.signInToStaging()

			_, ok := p.authConfig(host)
			assert.Assert(t, !ok)
			assert.Equal(t, p.connects.Load(), int32(0), "Docker Desktop must only be contacted for Docker Hub")
			assert.Equal(t, p.staging.sessionCalls.Load(), int32(0), "Docker Desktop must only be contacted for Docker Hub")
		})
	}
}

func TestDHIUsesDockerHubSession(t *testing.T) {
	for _, host := range []string{dhiRegistryHost, "https://" + dhiRegistryHost, "https://" + dhiRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub, nil)
			p.signInToStaging()

			want := desktopCredentials()
			want.ServerAddress = dhiRegistryHost
			got, ok := p.authConfig(host)
			assert.Assert(t, ok)
			assert.DeepEqual(t, got, want)
			assert.Equal(t, p.staging.sessionCalls.Load(), int32(0), "the staging session must not be read for DHI")

			got, ok = p.authConfig("docker.io")
			assert.Assert(t, ok)
			assert.DeepEqual(t, got, desktopCredentials())
			assert.Equal(t, hub.sessionCalls.Load(), int32(1))
		})
	}
}

func TestStagingRegistryUsesStagingSession(t *testing.T) {
	for _, host := range []string{stagingRegistryHost, "https://" + stagingRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
			p := newTestProvider(hub, nil)
			p.signInToStaging()

			got, ok := p.authConfig(host)
			assert.Assert(t, ok)
			assert.DeepEqual(t, got, stagingCredentials())
			assert.Equal(t, hub.sessionCalls.Load(), int32(0), "the production session must not be read for the staging registry")
		})
	}
}

func TestEnvironmentsDoNotMix(t *testing.T) {
	t.Run("staging session is not used for docker.io", func(t *testing.T) {
		p := newTestProvider(&fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile}, nil)
		p.signInToStaging()

		_, ok := p.authConfig("docker.io")
		assert.Assert(t, !ok)

		got, ok := p.authConfig(stagingRegistryHost)
		assert.Assert(t, ok)
		assert.DeepEqual(t, got, stagingCredentials())
	})

	t.Run("production session is not used for the staging registry", func(t *testing.T) {
		p := newTestProvider(&fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}, nil)

		_, ok := p.authConfig(stagingRegistryHost)
		assert.Assert(t, !ok)

		got, ok := p.authConfig("docker.io")
		assert.Assert(t, ok)
		assert.DeepEqual(t, got, desktopCredentials())
	})
}

func TestUsernameFromProfile(t *testing.T) {
	hub := &fakeHub{
		session: hubSession("", testNow.Add(time.Hour)),
		profile: dockerhub.Profile{Username: "hubuser"},
	}
	p := newTestProvider(hub, nil)

	got, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
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
			p := newTestProvider(tc.hub, nil)

			for range 2 {
				_, ok := p.authConfig("docker.io")
				assert.Assert(t, !ok)
			}
			assert.Equal(t, tc.hub.sessionCalls.Load(), int32(1), "Desktop must not be asked again within retryInterval")
		})
	}
}

func TestFallsBackWhenClientCannotBeCreated(t *testing.T) {
	p := newTestProvider(nil, errors.New("no socket path"))

	for range 2 {
		_, ok := p.authConfig("docker.io")
		assert.Assert(t, !ok)
	}
	assert.Equal(t, p.connects.Load(), int32(1))
}

func TestRetriesAfterInterval(t *testing.T) {
	hub := &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile}
	p := newTestProvider(hub, nil)

	_, ok := p.authConfig("docker.io")
	assert.Assert(t, !ok)

	hub.sessionErr = nil
	hub.session = hubSession("hubuser", testNow.Add(time.Hour))

	p.clock.advance(retryInterval - time.Second)
	_, ok = p.authConfig("docker.io")
	assert.Assert(t, !ok)
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	p.clock.advance(time.Second)
	got, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.DeepEqual(t, got, desktopCredentials())
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestReusesSessionUntilExpiry(t *testing.T) {
	expiresAt := testNow.Add(10 * time.Minute)
	hub := &fakeHub{session: hubSession("hubuser", expiresAt)}
	p := newTestProvider(hub, nil)

	_, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
	p.clock.advance(5 * time.Minute)
	_, ok = p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	hub.session = hubSession("hubuser", expiresAt.Add(time.Hour))
	hub.session.AccessToken = "refreshed-token"
	p.clock.advance(5*time.Minute - sessionExpirySkew)

	got, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.Equal(t, got.Password, "refreshed-token")
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestRefetchesSessionWithoutExpiry(t *testing.T) {
	hub := &fakeHub{session: hubSession("hubuser", time.Time{})}
	p := newTestProvider(hub, nil)

	_, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
	p.clock.advance(sessionTTL - time.Second)
	_, ok = p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))

	p.clock.advance(time.Second)
	_, ok = p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.Equal(t, hub.sessionCalls.Load(), int32(2))
}

func TestConcurrentLookupsShareOneFetch(t *testing.T) {
	hub := &fakeHub{session: hubSession("hubuser", testNow.Add(time.Hour))}
	p := newTestProvider(hub, nil)

	const lookups = 20
	results := make([]types.AuthConfig, lookups)
	var wg sync.WaitGroup
	for i := range lookups {
		wg.Go(func() {
			auth, ok := p.authConfig("docker.io")
			assert.Check(t, ok)
			results[i] = auth
		})
	}
	wg.Wait()

	for _, auth := range results {
		assert.DeepEqual(t, auth, desktopCredentials())
	}
	assert.Equal(t, hub.sessionCalls.Load(), int32(1))
	assert.Equal(t, p.connects.Load(), int32(1))
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
