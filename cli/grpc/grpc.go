// Package grpc establishes gRPC client connections to the Docker Engine's
// API endpoint.
//
// Alongside its HTTP API, the daemon serves gRPC services on the same
// endpoint; this is, for example, how BuildKit's control API is exposed.
// Daemons with API version 1.53 or later serve gRPC natively over HTTP/2:
// cleartext HTTP/2 (h2c) on non-TLS connections, and ALPN-negotiated HTTP/2
// on TLS connections. Older daemons only expose gRPC through the deprecated
// "POST /grpc" HTTP/1.1 upgrade endpoint.
//
// [Connect] hides those transport details: it returns a [grpc.ClientConn]
// reaching the daemon, whatever the transport of the current endpoint (unix
// or npipe socket, tcp with or without TLS, or a connection helper such as
// ssh://), falling back to the legacy upgrade endpoint for older daemons —
// or when an intermediary between the client and the daemon does not relay
// HTTP/2 even though the daemon's API version advertises native support
// (Docker Desktop's API proxy, at the time of writing). Note that the legacy
// endpoint only exposes the daemon's built-in gRPC services: services
// published by daemon extensions require a daemon serving gRPC natively.
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/docker/cli/cli/context/docker"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// nativeAPIVersion is the minimum API version where the daemon serves gRPC
// natively over HTTP/2 on its API endpoint. Older daemons only expose gRPC
// through the deprecated "POST /grpc" HTTP/1.1 upgrade endpoint.
const nativeAPIVersion = "1.53"

// defaultMaxMsgSize matches the message-size limits of the daemon's gRPC
// server.
const defaultMaxMsgSize = 16 << 20 // 16 MiB

// dummyTarget is the pseudo-target of connections established through a
// custom dialer; it only surfaces as the ":authority" header.
const dummyTarget = "docker-engine"

// APIClient is the subset of [client.APIClient] needed to establish a gRPC
// connection to the daemon.
type APIClient interface {
	Ping(ctx context.Context, options client.PingOptions) (client.PingResult, error)
	Dialer() func(context.Context) (net.Conn, error)
	DialHijack(ctx context.Context, path, proto string, meta map[string][]string) (net.Conn, error)
}

// DockerCLI is the subset of the docker CLI (command.Cli) needed to establish
// a gRPC connection to the daemon of its current endpoint.
type DockerCLI interface {
	Client() client.APIClient
	DockerEndpoint() docker.Endpoint
}

type config struct {
	dialMeta map[string][]string
	dialOpts []grpc.DialOption
}

// Opt customizes how the connection is established.
type Opt func(*config)

// WithDialMeta sets metadata attached to the connection request when the
// legacy upgrade endpoint is used. It has no effect on daemons serving gRPC
// natively; use per-call gRPC metadata instead.
func WithDialMeta(meta map[string][]string) Opt {
	return func(cfg *config) {
		cfg.dialMeta = meta
	}
}

// WithDialOptions appends gRPC dial options to the defaults used when
// establishing the connection.
func WithDialOptions(opts ...grpc.DialOption) Opt {
	return func(cfg *config) {
		cfg.dialOpts = append(cfg.dialOpts, opts...)
	}
}

// Connect returns a gRPC client connection to the daemon of dockerCLI's
// current endpoint. ctx bounds the daemon API version query and the probing
// of native connections; the returned connection is not bound to it, and
// behaves lazily afterwards.
func Connect(ctx context.Context, dockerCLI DockerCLI, opts ...Opt) (*grpc.ClientConn, error) {
	return Dial(ctx, dockerCLI.Client(), dockerCLI.DockerEndpoint(), opts...)
}

// Dial is like [Connect], for an API client and endpoint obtained separately.
func Dial(ctx context.Context, apiClient APIClient, ep docker.Endpoint, opts ...Opt) (*grpc.ClientConn, error) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	ping, err := apiClient.Ping(ctx, client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("establishing gRPC connection to the daemon: %w", err)
	}
	if ping.APIVersion != "" && !versions.LessThan(ping.APIVersion, nativeAPIVersion) {
		conn, err := dialNative(apiClient, ep, &cfg)
		if err != nil {
			return nil, err
		}
		if probeTransport(ctx, conn) == nil {
			return conn, nil
		}
		// The daemon's API version advertises native gRPC support but the
		// transport doesn't get through: an intermediary between the client
		// and the daemon (Docker Desktop's API proxy, at the time of writing)
		// may not relay HTTP/2. Fall back to the legacy upgrade endpoint.
		_ = conn.Close()
	}
	return dialLegacy(apiClient, &cfg)
}

// probeTransport verifies conn actually reaches an HTTP/2 server. Any
// RPC-level outcome — including Unimplemented from a daemon not serving the
// health service — proves the transport; only transport-level failures
// (codes.Unavailable) are reported.
func probeTransport(ctx context.Context, conn *grpc.ClientConn) error {
	_, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if status.Code(err) == codes.Unavailable {
		return err
	}
	return nil
}

// dialNative connects to a daemon serving gRPC natively on its API endpoint:
// ALPN-negotiated HTTP/2 for TLS endpoints, cleartext HTTP/2 (h2c) over the
// API client's own dialer otherwise. The client dialer covers unix and npipe
// sockets, plain tcp, and connection helpers such as ssh://.
func dialNative(apiClient APIClient, ep docker.Endpoint, cfg *config) (*grpc.ClientConn, error) {
	if strings.HasPrefix(ep.Host, "tcp://") {
		tlsCfg, err := ep.TLSConfig()
		if err != nil {
			return nil, err
		}
		if tlsCfg != nil {
			return dialTLS(ep.Host, tlsCfg, cfg)
		}
	}
	dialer := apiClient.Dialer()
	return grpc.NewClient("passthrough:///"+dummyTarget, dialOptions(cfg,
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return dialer(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)...)
}

// dialTLS connects to a TLS API endpoint. The daemon only accepts HTTP/2 on
// TLS connections through ALPN, so the TLS handshake is left to gRPC, which
// adds "h2" to the offered ALPN protocols.
func dialTLS(host string, tlsCfg *tls.Config, cfg *config) (*grpc.ClientConn, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(u.Host, dialOptions(cfg,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
	)...)
}

// dialLegacy connects through the deprecated "POST /grpc" HTTP/1.1 upgrade
// endpoint of daemons older than API v1.53: each connection upgrades an API
// request to a raw stream carrying cleartext HTTP/2, whatever the underlying
// transport. The upgrade request goes through the API client's regular
// plumbing, so it carries the configured dial metadata as headers.
func dialLegacy(apiClient APIClient, cfg *config) (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///"+dummyTarget, dialOptions(cfg,
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return apiClient.DialHijack(ctx, "/grpc", "h2c", cfg.dialMeta)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)...)
}

func dialOptions(cfg *config, opts ...grpc.DialOption) []grpc.DialOption {
	defaults := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(defaultMaxMsgSize),
			grpc.MaxCallSendMsgSize(defaultMaxMsgSize),
		),
	}
	return append(append(defaults, opts...), cfg.dialOpts...)
}
