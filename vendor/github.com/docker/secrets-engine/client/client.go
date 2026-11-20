// Copyright 2025-2026 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	healthv1 "github.com/docker/secrets-engine/client/api/health/v1"
	"github.com/docker/secrets-engine/client/api/health/v1/healthv1rpc"
	pluginsv1 "github.com/docker/secrets-engine/client/api/plugins/v1"
	"github.com/docker/secrets-engine/client/api/plugins/v1/pluginsv1rpc"
	"github.com/docker/secrets-engine/client/api/resolver"
	resolverv1 "github.com/docker/secrets-engine/client/api/resolver/v1"
	"github.com/docker/secrets-engine/client/api/resolver/v1/resolverv1rpc"
	clientconfig "github.com/docker/secrets-engine/client/config"
	"github.com/docker/secrets-engine/client/dockerhub"
	"github.com/docker/secrets-engine/client/rpc"
	"github.com/docker/secrets-engine/client/version"
	"github.com/docker/secrets-engine/secrets"
)

type (
	Envelope          = secrets.Envelope
	ID                = secrets.ID
	Pattern           = secrets.Pattern
	AuthorizeResponse = resolver.AuthorizeResponse
	DaemonVersion     = version.DaemonVersion
)

var (
	ParseID     = secrets.ParseID
	MustParseID = secrets.MustParseID

	ParsePattern     = secrets.ParsePattern
	MustParsePattern = secrets.MustParsePattern

	ErrSecretNotFound = secrets.ErrNotFound
	ErrAccessDenied   = secrets.ErrAccessDenied
)

var _ resolver.Resolver = &client{}

type Option func(c *config) error

func WithSocketPath(path string) Option {
	return func(s *config) error {
		if path == "" {
			return errors.New("no path provided")
		}
		if s.dialContext != nil {
			return errors.New("cannot set socket path and dial")
		}
		s.socketPath = path
		s.dialContext = dialFromPath(path)
		return nil
	}
}

func WithDialContext(dialContext func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(s *config) error {
		if s.dialContext != nil {
			return errors.New("cannot set socket path and dial")
		}
		s.dialContext = dialContext
		return nil
	}
}

// WithTimeout overrides the request timeout of the client.
//
// It is useful to set if there are hard-limits to when the client must wait
// for the server to accept the request.
//
// A timeout of 0 means no request timeout will be applied.
// Negative durations are not allowed and will result in an error.
func WithTimeout(timeout time.Duration) Option {
	return func(s *config) error {
		if timeout < 0 {
			return errors.New("request timeout duration cannot be negative")
		}
		s.requestTimeout = timeout
		return nil
	}
}

// WithResponseTimeout overrides the response header timeout of the client.
// By default no timeout is applied, because a request may block on user
// interaction. 0 means no timeout; negative durations return an error.
func WithResponseTimeout(responseTimeout time.Duration) Option {
	return func(s *config) error {
		if responseTimeout < 0 {
			return errors.New("response timeout duration cannot be negative")
		}
		s.responseTimeout = responseTimeout
		return nil
	}
}

type dial func(ctx context.Context, network, addr string) (net.Conn, error)

type config struct {
	socketPath      string
	dialContext     dial
	requestTimeout  time.Duration
	responseTimeout time.Duration
}

var (
	_ Client           = &client{}
	_ PluginManagement = &client{}
)

type client struct {
	resolverClient   resolverv1rpc.ResolverServiceClient
	authorizerClient resolverv1rpc.AuthorizerServiceClient
	versionClient    healthv1rpc.VersionServiceClient
	pluginsClient    pluginsv1rpc.PluginManagementServiceClient
}

// Authorize decides access to the patterns. The decision holds until the
// response expiry. A zero expiry means the decision never expires.
func (c client) Authorize(ctx context.Context, patterns ...secrets.Pattern) (resolver.AuthorizeResponse, error) {
	msgPatterns := make([]string, 0, len(patterns))
	for _, p := range patterns {
		msgPatterns = append(msgPatterns, p.String())
	}
	resp, err := c.authorizerClient.Authorize(ctx, resolverv1.AuthorizeRequest_builder{Patterns: msgPatterns}.Build())
	if err != nil {
		if rpc.CodeOf(err) == rpc.CodePermissionDenied {
			return resolver.AuthorizeResponse{}, secrets.ErrAccessDenied
		}
		return resolver.AuthorizeResponse{}, err
	}
	var expiry time.Time
	if resp.HasExpiresAt() {
		expiry = resp.GetExpiresAt().AsTime()
	}
	return resolver.AuthorizeResponse{
		Expiry: expiry,
		Allow:  resp.GetDecision() == resolverv1.Decision_DECISION_ALLOW,
	}, nil
}

func (c client) GetSecrets(ctx context.Context, pattern secrets.Pattern) ([]secrets.Envelope, error) {
	resp, err := c.resolverClient.GetSecrets(ctx, resolverv1.GetSecretsRequest_builder{Pattern: proto.String(pattern.String())}.Build())
	if err != nil {
		if rpc.CodeOf(err) == rpc.CodeNotFound {
			return nil, secrets.ErrNotFound
		}
		return nil, err
	}
	var envelopes []secrets.Envelope
	for _, item := range resp.GetEnvelopes() {
		id, err := secrets.ParseID(item.GetId())
		if err != nil {
			continue
		}
		envelopes = append(envelopes, secrets.Envelope{
			ID:         id,
			Value:      item.GetValue(),
			Metadata:   item.GetMetadata(),
			Provider:   item.GetProvider(),
			Version:    item.GetVersion(),
			CreatedAt:  item.GetCreatedAt().AsTime(),
			ResolvedAt: item.GetResolvedAt().AsTime(),
			ExpiresAt:  item.GetExpiresAt().AsTime(),
		})
	}
	return envelopes, nil
}

func (c client) HubAuth(opts ...dockerhub.Option) dockerhub.ClientAuth {
	return dockerhub.New(c, opts...)
}

func (c client) Version(ctx context.Context) (DaemonVersion, error) {
	resp, err := c.versionClient.GetVersion(ctx, healthv1.GetVersionRequest_builder{}.Build())
	if err != nil {
		return DaemonVersion{}, err
	}
	ver, err := version.NewVersion(resp.GetVersion())
	if err != nil {
		return DaemonVersion{}, fmt.Errorf("parsing daemon version %q: %w", resp.GetVersion(), err)
	}
	return DaemonVersion{Version: ver, Date: resp.GetDate(), CommitHash: resp.GetCommitHash()}, nil
}

// Client is the interface for interacting with the secrets engine daemon.
type Client interface {
	resolver.Resolver
	resolver.Authorizer

	// Version returns the name and version reported by the daemon.
	Version(ctx context.Context) (DaemonVersion, error)

	// HubAuth returns a Docker Hub authentication accessor backed by this client.
	HubAuth(opts ...dockerhub.Option) dockerhub.ClientAuth
}

type PluginManagement interface {
	ListPlugins(ctx context.Context) ([]PluginInfo, error)
	EnablePlugin(ctx context.Context, name string) error
	DisablePlugin(ctx context.Context, name string) error
}

func PluginManagementFromClient(c Client) (PluginManagement, error) {
	m, ok := c.(PluginManagement)
	if !ok {
		return nil, errors.New("client does not implement PluginManagement")
	}
	return m, nil
}

const baseURL = "http://unix"

// call sends a unary request in the Connect protocol:
// https://connectrpc.com/docs/protocol#unary-request
func call(rpcClient *rpc.Client, socketPath string) func(ctx context.Context, procedure string, req, resp proto.Message) error {
	return func(ctx context.Context, procedure string, req, resp proto.Message) error {
		err := rpcClient.Call(ctx, procedure, req, resp)
		if isDialError(err) {
			return connectError(socketPath, err)
		}
		return err
	}
}

func isDialError(err error) bool {
	if err == nil {
		return false
	}
	var oe *net.OpError
	if errors.As(err, &oe) && (oe.Op == "dial" || oe.Op == "connect") {
		return true
	}
	return false
}

func connectError(socketPath string, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return &ConnectError{Reason: connectReason(err), SocketPath: socketPath, Err: err}
}

// New creates a client that connects to config.StandaloneSocketPath by default.
// To connect to Docker Desktop, use WithSocketPath(config.DesktopSocketPath()).
func New(options ...Option) (Client, error) {
	cfg := &config{
		requestTimeout:  clientconfig.DefaultClientRequestTimeout,
		responseTimeout: clientconfig.DefaultClientResponseHeaderTimeout,
	}
	for _, opt := range options {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.dialContext == nil {
		cfg.socketPath = clientconfig.StandaloneSocketPath()
		cfg.dialContext = dialFromPath(cfg.socketPath)
	}
	c := &http.Client{
		Transport: &http.Transport{
			// re-use the same connection to the runtime, this speeds up subsequent
			// calls.
			MaxConnsPerHost:     clientconfig.DefaultClientMaxConnsPerHost,
			MaxIdleConnsPerHost: clientconfig.DefaultClientMaxIdleConnsPerHost,
			// keep the connection alive (good for long-lived clients)
			IdleConnTimeout: clientconfig.DefaultClientIdleConnTimeout,
			// no timeout by default; override with [WithResponseTimeout]
			ResponseHeaderTimeout: cfg.responseTimeout,
			TLSHandshakeTimeout:   clientconfig.DefaultClientTLSHandshakeTimeout,

			DialContext:        cfg.dialContext,
			DisableKeepAlives:  false,
			DisableCompression: false,
			ForceAttemptHTTP2:  true,
		},
		// by default Timeout will be 0 (meaning no timeout)
		// it can be overwritten with [WithTimeout]
		Timeout: cfg.requestTimeout,
	}
	callEngine := call(rpc.NewClient(c, baseURL), cfg.socketPath)
	return &client{
		resolverClient:   resolverv1rpc.NewResolverServiceClient(callEngine),
		authorizerClient: resolverv1rpc.NewAuthorizerServiceClient(callEngine),
		versionClient:    healthv1rpc.NewVersionServiceClient(callEngine),
		pluginsClient:    pluginsv1rpc.NewPluginManagementServiceClient(callEngine),
	}, nil
}

func (c client) ListPlugins(ctx context.Context) ([]PluginInfo, error) {
	resp, err := c.pluginsClient.ListPlugins(ctx, pluginsv1.ListPluginsRequest_builder{}.Build())
	if err != nil {
		return nil, err
	}
	var result []PluginInfo
	for _, item := range resp.GetPlugins() {
		name, err := version.NewName(item.GetName())
		if err != nil {
			continue
		}
		ver, err := version.NewVersion(item.GetVersion())
		if err != nil {
			continue
		}
		info := PluginInfo{
			Name:          name,
			Version:       ver,
			Disabled:      item.GetDisabled(),
			External:      item.GetExternal(),
			Configurable:  item.GetConfigurable(),
			RunStatus:     item.GetRunStatus(),
			StatusMessage: item.GetStatusMessage(),
		}
		switch item.WhichMetadata() {
		case pluginsv1.Plugin_SecretsProvider_case:
			pattern, err := secrets.ParsePattern(item.GetSecretsProvider().GetPattern())
			if err != nil {
				continue
			}
			info.SecretsProvider = &SecretsProviderMetadata{Pattern: pattern}
		case pluginsv1.Plugin_AccessControlModule_case:
			info.AccessControlModule = &AccessControlModuleMetadata{}
		}
		result = append(result, info)
	}
	return result, nil
}

func (c client) EnablePlugin(ctx context.Context, name string) error {
	r := pluginsv1.EnablePluginRequest_builder{}.Build()
	r.SetName(name)
	_, err := c.pluginsClient.EnablePlugin(ctx, r)
	return err
}

func (c client) DisablePlugin(ctx context.Context, name string) error {
	r := pluginsv1.DisablePluginRequest_builder{}.Build()
	r.SetName(name)
	_, err := c.pluginsClient.DisablePlugin(ctx, r)
	return err
}

type PluginInfo struct {
	Name                version.Name
	Version             version.Version
	Disabled            bool
	External            bool
	Configurable        bool
	RunStatus           pluginsv1.RunStatus
	StatusMessage       string
	SecretsProvider     *SecretsProviderMetadata
	AccessControlModule *AccessControlModuleMetadata
}

type AccessControlModuleMetadata struct{}

type SecretsProviderMetadata struct {
	Pattern secrets.Pattern
}

func dialFromPath(path string) dial {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := &net.Dialer{}
		return d.DialContext(ctx, "unix", path)
	}
}
