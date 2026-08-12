package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/cli/cli/context/docker"
	"github.com/moby/moby/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type fakeAPIClient struct {
	apiVersion string
	pingErr    error

	dialer func(context.Context) (net.Conn, error)
	hijack func(ctx context.Context, path, proto string, meta map[string][]string) (net.Conn, error)

	dialerCalled atomic.Bool
	hijackCalled atomic.Bool
	hijackURL    string
	hijackProto  string
	hijackMeta   map[string][]string
}

func (f *fakeAPIClient) Ping(context.Context, client.PingOptions) (client.PingResult, error) {
	return client.PingResult{APIVersion: f.apiVersion}, f.pingErr
}

func (f *fakeAPIClient) Dialer() func(context.Context) (net.Conn, error) {
	return func(ctx context.Context) (net.Conn, error) {
		f.dialerCalled.Store(true)
		return f.dialer(ctx)
	}
}

func (f *fakeAPIClient) DialHijack(ctx context.Context, path, proto string, meta map[string][]string) (net.Conn, error) {
	f.hijackCalled.Store(true)
	f.hijackURL = path
	f.hijackProto = proto
	f.hijackMeta = meta
	return f.hijack(ctx, path, proto, meta)
}

// startServer serves a gRPC health service on a loopback tcp socket, standing
// in for the daemon's API endpoint, and returns its address.
func startServer(t *testing.T, opts ...grpc.ServerOption) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	srv := grpc.NewServer(opts...)
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go func() {
		_ = srv.Serve(l)
	}()
	t.Cleanup(srv.Stop)
	return l.Addr().String()
}

func checkHealth(t *testing.T, conn *grpc.ClientConn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(resp.GetStatus(), healthpb.HealthCheckResponse_SERVING))
}

func TestDialNative(t *testing.T) {
	addr := startServer(t)
	apiClient := &fakeAPIClient{
		apiVersion: "1.53",
		dialer: func(context.Context) (net.Conn, error) {
			return net.Dial("tcp", addr)
		},
	}

	conn, err := Dial(t.Context(), apiClient, docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{Host: "unix:///var/run/docker.sock"},
	})
	assert.NilError(t, err)
	defer conn.Close()

	checkHealth(t, conn)
	assert.Check(t, apiClient.dialerCalled.Load())
	assert.Check(t, !apiClient.hijackCalled.Load())
}

// startHTTP1Server answers any connection with a plain HTTP/1.1 response,
// standing in for an intermediary that does not relay HTTP/2 (e.g. Docker
// Desktop's API proxy).
func startHTTP1Server(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				buf := make([]byte, 1024)
				_, _ = c.Read(buf)
				_, _ = c.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\nContent-Length: 0\r\n\r\n"))
				_ = c.Close()
			}(conn)
		}
	}()
	t.Cleanup(func() { _ = l.Close() })
	return l.Addr().String()
}

func TestDialNativeFallsBackToLegacy(t *testing.T) {
	// The daemon advertises native support but an HTTP/1.1-only intermediary
	// sits on the endpoint: the probe fails at transport level and the
	// connection falls back to the legacy upgrade endpoint.
	http1Addr := startHTTP1Server(t)
	grpcAddr := startServer(t)
	apiClient := &fakeAPIClient{
		apiVersion: "1.53",
		dialer: func(context.Context) (net.Conn, error) {
			return net.Dial("tcp", http1Addr)
		},
		hijack: func(context.Context, string, string, map[string][]string) (net.Conn, error) {
			return net.Dial("tcp", grpcAddr)
		},
	}

	conn, err := Dial(t.Context(), apiClient, docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{Host: "unix:///var/run/docker.sock"},
	})
	assert.NilError(t, err)
	defer conn.Close()

	checkHealth(t, conn)
	assert.Check(t, apiClient.dialerCalled.Load())
	assert.Check(t, apiClient.hijackCalled.Load())
	assert.Check(t, is.Equal(apiClient.hijackURL, "/grpc"))
}

func TestDialLegacy(t *testing.T) {
	for _, apiVersion := range []string{"", "1.52"} {
		t.Run("api-version-"+apiVersion, func(t *testing.T) {
			addr := startServer(t)
			meta := map[string][]string{"foo": {"bar"}}
			apiClient := &fakeAPIClient{
				apiVersion: apiVersion,
				hijack: func(context.Context, string, string, map[string][]string) (net.Conn, error) {
					return net.Dial("tcp", addr)
				},
			}

			conn, err := Dial(t.Context(), apiClient, docker.Endpoint{
				EndpointMeta: docker.EndpointMeta{Host: "unix:///var/run/docker.sock"},
			}, WithDialMeta(meta))
			assert.NilError(t, err)
			defer conn.Close()

			checkHealth(t, conn)
			assert.Check(t, apiClient.hijackCalled.Load())
			assert.Check(t, !apiClient.dialerCalled.Load())
			assert.Check(t, is.Equal(apiClient.hijackURL, "/grpc"))
			assert.Check(t, is.Equal(apiClient.hijackProto, "h2c"))
			assert.Check(t, is.DeepEqual(apiClient.hijackMeta, meta))
		})
	}
}

func TestDialTLS(t *testing.T) {
	cert := generateSelfSignedCert(t)
	addr := startServer(t, grpc.Creds(credentials.NewServerTLSFromCert(&cert)))
	apiClient := &fakeAPIClient{apiVersion: "1.53"}

	conn, err := Dial(t.Context(), apiClient, docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			Host: "tcp://" + addr,
			// Self-signed server certificate; TLS with ALPN is still
			// negotiated, which is what this test exercises.
			SkipTLSVerify: true,
		},
	})
	assert.NilError(t, err)
	defer conn.Close()

	checkHealth(t, conn)
	assert.Check(t, !apiClient.dialerCalled.Load())
	assert.Check(t, !apiClient.hijackCalled.Load())
}

func TestDialPingError(t *testing.T) {
	apiClient := &fakeAPIClient{pingErr: errors.New("daemon unreachable")}
	_, err := Dial(t.Context(), apiClient, docker.Endpoint{})
	assert.ErrorContains(t, err, "daemon unreachable")
}

func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NilError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	assert.NilError(t, err)
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
}
