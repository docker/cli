//go:build go1.26

package secretsengine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
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

	indexServer         = "https://index.docker.io/v1/"
	dhiRegistryHost     = "dhi.io"
	stagingRegistryHost = "registry-1-stage.docker.io"
)

var (
	errNoUsername     = errors.New("no username for the session")
	errSessionExpired = errors.New("the session has expired (not signed in to Docker Desktop?)")
)

type hubSessions interface {
	GetDefaultSession(ctx context.Context) (dockerhub.UserSession, error)
	GetDefaultProfile(ctx context.Context) (dockerhub.Profile, error)
}

var _ hubSessions = dockerhub.ClientAuth(nil)

type provider struct {
	hub     *session
	staging *session
}

var shared = &provider{
	hub:     newSession("Docker Hub"),
	staging: newSession("Docker Hub staging", dockerhub.Staging()),
}

var disabled atomic.Bool

func Disable() {
	disabled.Store(true)
}

func AuthConfig(registryHostname string) (types.AuthConfig, bool) {
	if disabled.Load() {
		return types.AuthConfig{}, false
	}
	return shared.authConfig(registryHostname)
}

func (p *provider) authConfig(registryHostname string) (types.AuthConfig, bool) {
	s, serverAddress := p.sessionFor(registryHostname)
	if s == nil {
		return types.AuthConfig{}, false
	}
	auth, ok := s.auth()
	if !ok {
		return types.AuthConfig{}, false
	}
	auth.ServerAddress = serverAddress
	return auth, true
}

func (p *provider) sessionFor(registryHostname string) (*session, string) {
	switch credentials.ConvertToHostname(registryHostname) {
	case "docker.io", "index.docker.io", "registry-1.docker.io":
		return p.hub, indexServer
	case dhiRegistryHost:
		return p.hub, dhiRegistryHost
	case stagingRegistryHost:
		return p.staging, stagingRegistryHost
	default:
		return nil, ""
	}
}

type session struct {
	name    string
	connect func() (hubSessions, error)
	timeout time.Duration
	now     func() time.Time

	mu         sync.Mutex
	hub        hubSessions
	retryAt    time.Time
	cached     *types.AuthConfig
	validUntil time.Time
}

func newSession(name string, opts ...dockerhub.Option) *session {
	return &session{
		name: name,
		connect: func() (hubSessions, error) {
			return connect(api.DesktopSocketPath(), opts...)
		},
		timeout: lookupTimeout,
		now:     time.Now,
	}
}

func connect(socketPath string, opts ...dockerhub.Option) (hubSessions, error) {
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
