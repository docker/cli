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
	"testing/synctest"
	"time"

	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/registry"
	seclient "github.com/docker/secrets-engine/client"
	"github.com/docker/secrets-engine/client/dockerhub"
	"gotest.tools/v3/assert"
)

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

func hubSession(username string, expiresAt time.Time) dockerhub.UserSession {
	s := dockerhub.UserSession{AccessToken: "desktop-oauth-token"}
	s.Claims.Username = username
	if !expiresAt.IsZero() {
		s.Claims.ExpiresAt = &dockerhub.NumericDate{Time: expiresAt}
	}
	return s
}

func inAnHour() time.Time {
	return time.Now().Add(time.Hour)
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
	return types.AuthConfig{Username: "stageuser", Password: "staging-oauth-token", ServerAddress: stagingRegistryHost}
}

type testProvider struct {
	*provider
	fallback *recordingFallback
	staging  *fakeHub
}

func newTestProvider(hub hubClient) *testProvider {
	tp := &testProvider{
		fallback: &recordingFallback{},
		staging:  &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile},
	}
	tp.provider = &provider{
		fallback: tp.fallback,
		hub:      &session{name: "Docker Hub", hub: hub},
		staging:  &session{name: "Docker Hub staging", hub: tp.staging},
	}
	return tp
}

func (tp *testProvider) signInToStaging() {
	tp.staging.sessionErr = nil
	tp.staging.session = hubSession("stageuser", inAnHour())
	tp.staging.session.AccessToken = "staging-oauth-token"
}

func connectError(reason seclient.ConnectReason) error {
	return &seclient.ConnectError{
		Reason:     reason,
		SocketPath: "/run/engine.sock",
		Err:        errors.New("dial unix /run/engine.sock"),
	}
}

func TestNewDesktopUsesSeparateSessions(t *testing.T) {
	t.Parallel()
	p := NewDesktop(&recordingFallback{}).(*provider)
	assert.Assert(t, p.hub != p.staging, "production and staging sessions must be separate")
}

func TestDockerHubUsesDesktopSession(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"docker.io", "index.docker.io", "registry-1.docker.io", registry.IndexServer} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			hub := &fakeHub{session: hubSession("hubuser", inAnHour())}
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
	t.Parallel()
	hosts := []string{
		"ghcr.io", "localhost:5000", "registry.example.com", "",
		stagingRegistryHost + ".example.com", "example.com/" + stagingRegistryHost,
		dhiRegistryHost + ".example.com", "example.com/" + dhiRegistryHost, "registry." + dhiRegistryHost,
		"docker.io.example.com",
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			hub := &fakeHub{session: hubSession("hubuser", inAnHour())}
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
	t.Parallel()
	for _, host := range []string{dhiRegistryHost, "https://" + dhiRegistryHost, "https://" + dhiRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			hub := &fakeHub{session: hubSession("hubuser", inAnHour())}
			p := newTestProvider(hub)
			p.signInToStaging()

			want := desktopCredentials()
			want.ServerAddress = dhiRegistryHost
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
	t.Parallel()
	for _, host := range []string{stagingRegistryHost, "https://" + stagingRegistryHost + "/"} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			hub := &fakeHub{session: hubSession("hubuser", inAnHour())}
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
	t.Parallel()
	t.Run("staging session is not used for docker.io", func(t *testing.T) {
		t.Parallel()
		p := newTestProvider(&fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile})
		p.signInToStaging()

		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))

		got, err = p.GetAuthConfig(stagingRegistryHost)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, stagingCredentials())
	})

	t.Run("production session is not used for the staging registry", func(t *testing.T) {
		t.Parallel()
		p := newTestProvider(&fakeHub{session: hubSession("hubuser", inAnHour())})

		got, err := p.GetAuthConfig(stagingRegistryHost)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials(stagingRegistryHost))

		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
	})
}

func TestUsernameFromProfile(t *testing.T) {
	t.Parallel()
	hub := &fakeHub{
		session: hubSession("", inAnHour()),
		profile: dockerhub.Profile{Username: "hubuser"},
	}
	p := newTestProvider(hub)

	got, err := p.GetAuthConfig("docker.io")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, desktopCredentials())
}

func TestFallsBackWithoutUsableSession(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		hub  func() *fakeHub
	}{
		{name: "secrets engine not running", hub: func() *fakeHub { return &fakeHub{sessionErr: connectError(seclient.ReasonNotRunning)} }},
		{name: "secrets engine socket not accessible", hub: func() *fakeHub { return &fakeHub{sessionErr: connectError(seclient.ReasonPermissionDenied)} }},
		{name: "nobody signed in", hub: func() *fakeHub { return &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile} }},
		{name: "access denied", hub: func() *fakeHub { return &fakeHub{sessionErr: seclient.ErrAccessDenied} }},
		{name: "no username", hub: func() *fakeHub { return &fakeHub{session: hubSession("", inAnHour())} }},
		{name: "profile lookup fails", hub: func() *fakeHub {
			return &fakeHub{session: hubSession("", inAnHour()), profileErr: errors.New("boom")}
		}},
		{name: "expired", hub: func() *fakeHub { return &fakeHub{session: hubSession("hubuser", time.Now().Add(-time.Minute))} }},
		{name: "expires within the skew", hub: func() *fakeHub {
			return &fakeHub{session: hubSession("hubuser", time.Now().Add(sessionExpirySkew/2))}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				hub := tc.hub()
				p := newTestProvider(hub)

				for range 2 {
					got, err := p.GetAuthConfig("docker.io")
					assert.NilError(t, err)
					assert.DeepEqual(t, got, cliCredentials("docker.io"))
				}
				assert.Equal(t, hub.sessionCalls.Load(), int32(1), "Desktop must not be asked again within retryInterval")
			})
		})
	}
}

func TestDesktopFallsBackWhenClientCannotBeCreated(t *testing.T) {
	t.Parallel()
	var connects atomic.Int32
	s := &session{
		name: "Docker Hub",
		connect: func() (hubClient, error) {
			connects.Add(1)
			return nil, errors.New("no socket path")
		},
	}
	p := &provider{fallback: &recordingFallback{}, hub: s}

	for range 2 {
		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))
	}
	assert.Equal(t, connects.Load(), int32(1))
}

func TestRetriesAfterInterval(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		hub := &fakeHub{sessionErr: dockerhub.ErrNoDefaultProfile}
		p := newTestProvider(hub)

		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))

		hub.sessionErr = nil
		hub.session = hubSession("hubuser", inAnHour())

		time.Sleep(retryInterval - time.Second)
		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, cliCredentials("docker.io"))
		assert.Equal(t, hub.sessionCalls.Load(), int32(1))

		time.Sleep(time.Second)
		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		assert.Equal(t, hub.sessionCalls.Load(), int32(2))
	})
}

func TestReusesSessionUntilExpiry(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		expiresAt := time.Now().Add(10 * time.Minute)
		hub := &fakeHub{session: hubSession("hubuser", expiresAt)}
		p := newTestProvider(hub)

		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		time.Sleep(5 * time.Minute)
		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		assert.Equal(t, hub.sessionCalls.Load(), int32(1))

		hub.session = hubSession("hubuser", expiresAt.Add(time.Hour))
		hub.session.AccessToken = "refreshed-token"
		time.Sleep(5*time.Minute - sessionExpirySkew)

		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.Equal(t, got.Password, "refreshed-token")
		assert.Equal(t, hub.sessionCalls.Load(), int32(2))
	})
}

func TestRefetchesSessionWithoutExpiry(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		hub := &fakeHub{session: hubSession("hubuser", time.Time{})}
		p := newTestProvider(hub)

		got, err := p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		time.Sleep(sessionTTL - time.Second)
		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		assert.Equal(t, hub.sessionCalls.Load(), int32(1))

		time.Sleep(time.Second)
		got, err = p.GetAuthConfig("docker.io")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, desktopCredentials())
		assert.Equal(t, hub.sessionCalls.Load(), int32(2))
	})
}

func TestConcurrentLookupsShareOneFetch(t *testing.T) {
	t.Parallel()
	hub := &fakeHub{session: hubSession("hubuser", inAnHour())}
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
	t.Parallel()
	_, err := connect(filepath.Join(t.TempDir(), "engine.sock"))
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), err)
	assert.Assert(t, isSessionUnavailable(err))
}

func TestIsSessionUnavailable(t *testing.T) {
	t.Parallel()
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
		errors.New("no username for the session"),
		context.DeadlineExceeded,
	} {
		assert.Assert(t, !isSessionUnavailable(err), err)
	}
}
