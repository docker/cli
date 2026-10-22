//go:build go1.26

package secretsengine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	seclient "github.com/docker/secrets-engine/client"
	"github.com/docker/secrets-engine/client/dockerhub"
	"github.com/docker/secrets-engine/x/api/resolver"
	"github.com/docker/secrets-engine/x/api/resolver/v1/resolverv1connect"
	"github.com/docker/secrets-engine/x/secrets"
	"gotest.tools/v3/assert"
)

const benchSocketEnv = "SECRETSENGINE_BENCH_SOCKET"

type fakeResolver struct {
	latency time.Duration
	values  map[string][]byte
	calls   atomic.Int64
}

func (r *fakeResolver) GetSecrets(ctx context.Context, pattern secrets.Pattern) ([]secrets.Envelope, error) {
	r.calls.Add(1)
	if r.latency > 0 {
		select {
		case <-time.After(r.latency):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	var envelopes []secrets.Envelope
	for key, value := range r.values {
		id := secrets.MustParseID(key)
		if pattern.Match(id) {
			envelopes = append(envelopes, secrets.Envelope{ID: id, Value: value, ResolvedAt: time.Now()})
		}
	}
	if len(envelopes) == 0 {
		return nil, secrets.ErrNotFound
	}
	return envelopes, nil
}

func fakeAccessToken() string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + enc([]byte(`{"sub":"bench"}`)) + "." + enc([]byte("signature"))
}

func signedIn(tb testing.TB, expiresAt time.Time) map[string][]byte {
	tb.Helper()
	const username = "hubuser"
	userID := "docker/auth/hub/" + username
	profile, err := json.Marshal(dockerhub.Profile{UserID: userID, Username: username, Email: username + "@example.com"})
	assert.NilError(tb, err)
	s := dockerhub.UserSession{AccessToken: fakeAccessToken()}
	s.Claims.Username = username
	s.Claims.ExpiresAt = &dockerhub.NumericDate{Time: expiresAt}
	userSession, err := json.Marshal(s)
	assert.NilError(tb, err)
	return map[string][]byte{
		"docker/auth/metadata/hub/default": profile,
		userID:                             userSession,
	}
}

var socketDirs atomic.Int64

func socketDir(tb testing.TB) string {
	tb.Helper()
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("se-%d-%d", os.Getpid(), socketDirs.Add(1)))
	assert.NilError(tb, os.Mkdir(dir, 0o700))
	tb.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

type fakeEngine struct {
	socketPath string

	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func startEngine(tb testing.TB, r *fakeResolver) *fakeEngine {
	tb.Helper()
	e := &fakeEngine{
		socketPath: filepath.Join(socketDir(tb), "engine.sock"),
		conns:      map[net.Conn]struct{}{},
	}
	ln, err := net.Listen("unix", e.socketPath)
	assert.NilError(tb, err)
	mux := http.NewServeMux()
	mux.Handle(resolverv1connect.NewResolverServiceHandler(resolver.NewResolverHandler(r)))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ConnState: e.track}
	go func() { _ = srv.Serve(ln) }()
	tb.Cleanup(func() { _ = srv.Close() })
	return e
}

func (e *fakeEngine) track(c net.Conn, state http.ConnState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch state {
	case http.StateNew:
		e.conns[c] = struct{}{}
	case http.StateClosed, http.StateHijacked:
		delete(e.conns, c)
	case http.StateActive, http.StateIdle:
	}
}

func (e *fakeEngine) dropConnections() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for c := range e.conns {
		_ = c.Close()
	}
	clear(e.conns)
}

func startHungEngine(tb testing.TB) string {
	tb.Helper()
	socketPath := filepath.Join(socketDir(tb), "engine.sock")
	ln, err := net.Listen("unix", socketPath)
	assert.NilError(tb, err)
	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			held = append(held, c)
			mu.Unlock()
		}
	}()
	tb.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range held {
			_ = c.Close()
		}
	})
	return socketPath
}

func staleSocket(tb testing.TB) string {
	tb.Helper()
	socketPath := filepath.Join(socketDir(tb), "engine.sock")
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	assert.NilError(tb, err)
	ln.SetUnlinkOnClose(false)
	assert.NilError(tb, ln.Close())
	_, err = os.Lstat(socketPath)
	assert.NilError(tb, err)
	return socketPath
}

func socketSession(socketPath string, timeout time.Duration) *session {
	return &session{
		name: "Docker Hub",
		connect: func() (hubSessions, error) {
			return connect(socketPath)
		},
		timeout: timeout,
		now:     time.Now,
	}
}

func TestEngineSignedIn(t *testing.T) {
	r := &fakeResolver{values: signedIn(t, time.Now().Add(time.Hour))}
	e := startEngine(t, r)
	p := &provider{hub: socketSession(e.socketPath, lookupTimeout)}

	got, ok := p.authConfig("docker.io")
	assert.Assert(t, ok)
	assert.Equal(t, got.Username, "hubuser")
	assert.Equal(t, got.Password, fakeAccessToken())
	assert.Equal(t, got.ServerAddress, indexServer)
	assert.Equal(t, got.IdentityToken, "")

	got, ok = p.authConfig("dhi.io")
	assert.Assert(t, ok)
	assert.Equal(t, got.ServerAddress, dhiRegistryHost)
	assert.Equal(t, r.calls.Load(), int64(2), "the profile and the session are read once, then cached")
}

func TestEngineSignedOut(t *testing.T) {
	r := &fakeResolver{}
	e := startEngine(t, r)
	p := &provider{hub: socketSession(e.socketPath, lookupTimeout)}

	_, ok := p.authConfig("docker.io")
	assert.Assert(t, !ok)
	assert.Equal(t, r.calls.Load(), int64(1))
}

func TestMissingSocketFallsBackImmediately(t *testing.T) {
	s := socketSession(filepath.Join(socketDir(t), "engine.sock"), lookupTimeout)

	start := time.Now()
	_, ok := s.auth()
	assert.Assert(t, !ok)
	assert.Assert(t, time.Since(start) < time.Second)
	assert.Assert(t, s.hub == nil, "no client is created without a socket")
}

func TestStaleSocketFallsBackImmediately(t *testing.T) {
	s := socketSession(staleSocket(t), lookupTimeout)

	start := time.Now()
	_, ok := s.auth()
	assert.Assert(t, !ok)
	assert.Assert(t, time.Since(start) < time.Second)
}

func TestHungEngineIsBounded(t *testing.T) {
	const timeout = 200 * time.Millisecond
	s := socketSession(startHungEngine(t), timeout)

	start := time.Now()
	_, ok := s.auth()
	elapsed := time.Since(start)
	assert.Assert(t, !ok)
	assert.Assert(t, elapsed >= timeout, elapsed)
	assert.Assert(t, elapsed < timeout+2*time.Second, elapsed)
}

func benchmarkColdLookup(b *testing.B, socketPath string, e *fakeEngine, want bool) {
	b.Helper()
	b.ReportAllocs()
	for range b.N {
		s := socketSession(socketPath, lookupTimeout)
		if _, ok := s.auth(); ok != want {
			b.Fatalf("got a Desktop session: %t, want %t", ok, want)
		}
		if e != nil {
			b.StopTimer()
			e.dropConnections()
			b.StartTimer()
		}
	}
}

func BenchmarkAuthConfig(b *testing.B) {
	b.Run("other-registry", func(b *testing.B) {
		p := &provider{hub: socketSession(filepath.Join(socketDir(b), "engine.sock"), lookupTimeout)}
		b.ReportAllocs()
		for range b.N {
			if _, ok := p.authConfig("ghcr.io"); ok {
				b.Fatal("unexpected Desktop session")
			}
		}
	})
	b.Run("missing-socket", func(b *testing.B) {
		benchmarkColdLookup(b, filepath.Join(socketDir(b), "engine.sock"), nil, false)
	})
	b.Run("missing-socket-without-stat", func(b *testing.B) {
		socketPath := filepath.Join(socketDir(b), "engine.sock")
		b.ReportAllocs()
		for range b.N {
			c, err := seclient.New(seclient.WithSocketPath(socketPath), seclient.WithTimeout(lookupTimeout))
			if err != nil {
				b.Fatal(err)
			}
			if _, err := c.HubAuth().GetDefaultSession(context.Background()); err == nil {
				b.Fatal("unexpected Desktop session")
			}
		}
	})
	b.Run("stale-socket", func(b *testing.B) {
		benchmarkColdLookup(b, staleSocket(b), nil, false)
	})
	b.Run("signed-out", func(b *testing.B) {
		e := startEngine(b, &fakeResolver{})
		benchmarkColdLookup(b, e.socketPath, e, false)
	})
	b.Run("signed-in", func(b *testing.B) {
		e := startEngine(b, &fakeResolver{values: signedIn(b, time.Now().Add(time.Hour))})
		benchmarkColdLookup(b, e.socketPath, e, true)
	})
	b.Run("signed-in-10ms-engine", func(b *testing.B) {
		e := startEngine(b, &fakeResolver{latency: 10 * time.Millisecond, values: signedIn(b, time.Now().Add(time.Hour))})
		benchmarkColdLookup(b, e.socketPath, e, true)
	})
	b.Run("signed-in-refetch", func(b *testing.B) {
		e := startEngine(b, &fakeResolver{values: signedIn(b, time.Now().Add(time.Hour))})
		s := socketSession(e.socketPath, lookupTimeout)
		if _, ok := s.auth(); !ok {
			b.Fatal("no Desktop session")
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			s.cached = nil
			if _, ok := s.auth(); !ok {
				b.Fatal("no Desktop session")
			}
		}
	})
	b.Run("signed-in-cached", func(b *testing.B) {
		e := startEngine(b, &fakeResolver{values: signedIn(b, time.Now().Add(time.Hour))})
		s := socketSession(e.socketPath, lookupTimeout)
		if _, ok := s.auth(); !ok {
			b.Fatal("no Desktop session")
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, ok := s.auth(); !ok {
				b.Fatal("no Desktop session")
			}
		}
	})
}

type trackingDialer struct {
	mu    sync.Mutex
	conns []net.Conn
}

func (d *trackingDialer) dial(socketPath string) func(ctx context.Context, _, _ string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		var nd net.Dialer
		c, err := nd.DialContext(ctx, "unix", socketPath)
		if err != nil {
			return nil, err
		}
		d.mu.Lock()
		d.conns = append(d.conns, c)
		d.mu.Unlock()
		return c, nil
	}
}

func (d *trackingDialer) closeAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.conns {
		_ = c.Close()
	}
	d.conns = nil
}

func BenchmarkExternalEngine(b *testing.B) {
	socketPath := os.Getenv(benchSocketEnv)
	if socketPath == "" {
		b.Skipf("%s is not set", benchSocketEnv)
	}
	b.Run("cold", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var dialer trackingDialer
			s := &session{
				name: "Docker Hub",
				connect: func() (hubSessions, error) {
					c, err := seclient.New(seclient.WithDialContext(dialer.dial(socketPath)), seclient.WithTimeout(lookupTimeout))
					if err != nil {
						return nil, err
					}
					return c.HubAuth(), nil
				},
				timeout: lookupTimeout,
				now:     time.Now,
			}
			_, ok := s.auth()
			b.StopTimer()
			dialer.closeAll()
			if !ok {
				b.Fatal("no Docker Hub session from the engine at " + socketPath)
			}
			b.StartTimer()
		}
	})
	b.Run("refetch", func(b *testing.B) {
		s := socketSession(socketPath, lookupTimeout)
		if _, ok := s.auth(); !ok {
			b.Fatal("no Docker Hub session from the engine at " + socketPath)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			s.cached = nil
			if _, ok := s.auth(); !ok {
				b.Fatal("no Docker Hub session from the engine at " + socketPath)
			}
		}
	})
}
