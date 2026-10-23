//go:build go1.26

package secretsengine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/registry"
	seclient "github.com/docker/secrets-engine/client"
	"github.com/docker/secrets-engine/client/dockerhub"
	"github.com/docker/secrets-engine/x/api"
	"github.com/sirupsen/logrus"
)

const (
	lookupTimeout     = 5 * time.Second
	sessionExpirySkew = 30 * time.Second
	sessionTTL        = time.Minute
	retryInterval     = time.Minute
)

var (
	errNoUsername     = errors.New("no username for the session")
	errSessionExpired = errors.New("the session has expired (not signed in to Docker Desktop?)")
)

type hubClient interface {
	GetDefaultSession(ctx context.Context) (dockerhub.UserSession, error)
	GetDefaultProfile(ctx context.Context) (dockerhub.Profile, error)
}

var _ hubClient = dockerhub.ClientAuth(nil)

type AuthProvider interface {
	GetAuthConfig(registryHostname string) (types.AuthConfig, error)
}

type provider struct {
	fallback AuthProvider
	hub      *session
	staging  *session
}

func newProvider(fallback AuthProvider, hub, staging hubClient) *provider {
	return &provider{
		fallback: fallback,
		hub:      newSession("Docker Hub", hub),
		staging:  newSession("Docker Hub staging", staging),
	}
}

var (
	desktopHubSession     = newDesktopSession("Docker Hub")
	desktopStagingSession = newDesktopSession("Docker Hub staging", dockerhub.Staging())
)

func NewDesktop(fallback AuthProvider) AuthProvider {
	return &provider{
		fallback: fallback,
		hub:      desktopHubSession,
		staging:  desktopStagingSession,
	}
}

func (p *provider) GetAuthConfig(registryHostname string) (types.AuthConfig, error) {
	if s, serverAddress := p.sessionFor(registryHostname); s != nil {
		if auth, ok := s.auth(); ok {
			auth.ServerAddress = serverAddress
			return auth, nil
		}
	}
	return p.fallback.GetAuthConfig(registryHostname)
}

func (p *provider) sessionFor(registryHostname string) (*session, string) {
	switch credentials.ConvertToHostname(registryHostname) {
	case registry.IndexName, registry.IndexHostname, registry.DefaultRegistryHost:
		return p.hub, registry.IndexServer
	case registry.DHIRegistryHost:
		return p.hub, registry.DHIRegistryHost
	case registry.StagingRegistryHost:
		return p.staging, registry.StagingRegistryHost
	default:
		return nil, ""
	}
}

type session struct {
	name    string
	connect func() (hubClient, error)
	timeout time.Duration
	now     func() time.Time

	mu         sync.Mutex
	hub        hubClient
	retryAt    time.Time
	cached     *types.AuthConfig
	validUntil time.Time
}

func newSession(name string, hub hubClient) *session {
	return &session{
		name:    name,
		hub:     hub,
		timeout: lookupTimeout,
		now:     time.Now,
	}
}

func newDesktopSession(name string, opts ...dockerhub.Option) *session {
	s := newSession(name, nil)
	s.connect = func() (hubClient, error) {
		return connect(api.DesktopSocketPath(), opts...)
	}
	return s
}

func connect(socketPath string, opts ...dockerhub.Option) (hubClient, error) {
	if _, err := os.Lstat(socketPath); errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	c, err := seclient.New(
		seclient.WithSocketPath(socketPath),
		seclient.WithTimeout(lookupTimeout),
	)
	if err != nil {
		return nil, err
	}
	return c.HubAuth(opts...), nil
}

func (s *session) auth() (types.AuthConfig, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if now.Before(s.retryAt) {
		return types.AuthConfig{}, false
	}
	if s.cached != nil && now.Before(s.validUntil) {
		return *s.cached, true
	}
	auth, validUntil, err := s.fetch()
	if err != nil {
		s.retryAt = s.now().Add(retryInterval)
		s.cached = nil
		logFallback(s.name, err)
		return types.AuthConfig{}, false
	}
	logrus.Debugf("using the %s session of %q from Docker Desktop", s.name, auth.Username)
	s.cached, s.validUntil = &auth, validUntil
	return auth, true
}

func (s *session) fetch() (types.AuthConfig, time.Time, error) {
	if s.hub == nil {
		hub, err := s.connect()
		if err != nil {
			return types.AuthConfig{}, time.Time{}, err
		}
		s.hub = hub
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	userSession, err := s.hub.GetDefaultSession(ctx)
	if err != nil {
		return types.AuthConfig{}, time.Time{}, err
	}

	username := userSession.Claims.Username
	if username == "" {
		profile, err := s.hub.GetDefaultProfile(ctx)
		if err != nil {
			return types.AuthConfig{}, time.Time{}, fmt.Errorf("resolving the Docker Hub username: %w", err)
		}
		username = profile.Username
	}
	if username == "" {
		return types.AuthConfig{}, time.Time{}, errNoUsername
	}

	now := s.now()
	validUntil := now.Add(sessionTTL)
	if exp := userSession.Claims.ExpiresAt; exp != nil {
		validUntil = exp.Add(-sessionExpirySkew)
		if !now.Before(validUntil) {
			return types.AuthConfig{}, time.Time{}, errSessionExpired
		}
	}

	return types.AuthConfig{
		Username: username,
		Password: userSession.AccessToken,
	}, validUntil, nil
}

func logFallback(name string, err error) {
	if isSessionUnavailable(err) {
		logrus.Debugf("no %s session available from Docker Desktop, using Docker CLI credentials: %v", name, err)
		return
	}
	logrus.Warnf("could not use the %s session from Docker Desktop, using Docker CLI credentials instead: %v", name, err)
}

func isSessionUnavailable(err error) bool {
	if _, ok := errors.AsType[*seclient.ConnectError](err); ok {
		return true
	}
	return errors.Is(err, fs.ErrNotExist) ||
		errors.Is(err, dockerhub.ErrNoSession) ||
		errors.Is(err, errSessionExpired)
}
