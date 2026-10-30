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

	"connectrpc.com/connect"

	"github.com/docker/secrets-engine/client/dockerhub"
	"github.com/docker/secrets-engine/x/api"
	healthv1 "github.com/docker/secrets-engine/x/api/health/v1"
	"github.com/docker/secrets-engine/x/api/health/v1/healthv1connect"
	pluginsv1 "github.com/docker/secrets-engine/x/api/plugins/v1"
	"github.com/docker/secrets-engine/x/api/plugins/v1/pluginsv1connect"
	"github.com/docker/secrets-engine/x/api/resolver"
	"github.com/docker/secrets-engine/x/secrets"
)

type (
	Envelope          = secrets.Envelope
	ID                = secrets.ID
	Pattern           = secrets.Pattern
	AuthorizeResponse = secrets.AuthorizeResponse
	DaemonVersion     = api.DaemonVersion
)

var (
	ParseID     = secrets.ParseID
	MustParseID = secrets.MustParseID

	ParsePattern     = secrets.ParsePattern
	MustParsePattern = secrets.MustParsePattern

	ErrSecretNotFound = secrets.ErrNotFound
	ErrAccessDenied   = secrets.ErrAccessDenied
)

var _ secrets.Resolver = &client{}

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
	socketPath       string
	resolverClient   secrets.Resolver
	engineClient     pluginsv1connect.PluginManagementServiceClient
	versionClient    healthv1connect.VersionServiceClient
	authorizerClient secrets.Authorizer
}

// Authorize decides access to the patterns. The decision holds until the
// response expiry. A zero expiry means the decision never expires.
func (c client) Authorize(ctx context.Context, patterns ...secrets.Pattern) (secrets.AuthorizeResponse, error) {
	resp, err := c.authorizerClient.Authorize(ctx, patterns...)
	if isDialError(err) {
		return secrets.AuthorizeResponse{}, c.connectError(err)
	}
	if err != nil {
		return secrets.AuthorizeResponse{}, err
	}
	return resp, nil
}

func (c client) GetSecrets(ctx context.Context, pattern secrets.Pattern) ([]secrets.Envelope, error) {
	envelopes, err := c.resolverClient.GetSecrets(ctx, pattern)
	if isDialError(err) {
		return nil, c.connectError(err)
	}
	if err != nil {
		return nil, err
	}
	return envelopes, nil
}

func (c client) HubAuth(opts ...dockerhub.Option) dockerhub.ClientAuth {
	return dockerhub.New(c, opts...)
}

func (c client) Version(ctx context.Context) (DaemonVersion, error) {
	resp, err := c.versionClient.GetVersion(ctx, connect.NewRequest(healthv1.GetVersionRequest_builder{}.Build()))
	if isDialError(err) {
		return DaemonVersion{}, c.connectError(err)
	}
	if err != nil {
		return DaemonVersion{}, err
	}
	ver, err := api.NewVersion(resp.Msg.GetVersion())
	if err != nil {
		return DaemonVersion{}, fmt.Errorf("parsing daemon version %q: %w", resp.Msg.GetVersion(), err)
	}
	return DaemonVersion{Version: ver, Date: resp.Msg.GetDate(), CommitHash: resp.Msg.GetCommitHash()}, nil
}

// Client is the interface for interacting with the secrets engine daemon.
type Client interface {
	secrets.Resolver
	secrets.Authorizer

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

func (c client) connectError(err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return &ConnectError{Reason: connectReason(err), SocketPath: c.socketPath, Err: err}
}

// New creates a client that connects to [api.StandaloneSocketPath] by default.
// To connect to Docker Desktop, use WithSocketPath(api.DesktopSocketPath()).
func New(options ...Option) (Client, error) {
	cfg := &config{
		requestTimeout:  api.DefaultClientRequestTimeout,
		responseTimeout: api.DefaultClientResponseHeaderTimeout,
	}
	for _, opt := range options {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.dialContext == nil {
		cfg.socketPath = api.StandaloneSocketPath()
		cfg.dialContext = dialFromPath(cfg.socketPath)
	}
	c := &http.Client{
		Transport: &http.Transport{
			// re-use the same connection to the runtime, this speeds up subsequent
			// calls.
			MaxConnsPerHost:     api.DefaultClientMaxConnsPerHost,
			MaxIdleConnsPerHost: api.DefaultClientMaxIdleConnsPerHost,
			// keep the connection alive (good for long-lived clients)
			IdleConnTimeout: api.DefaultClientIdleConnTimeout,
			// no timeout by default; override with [WithResponseTimeout]
			ResponseHeaderTimeout: cfg.responseTimeout,
			TLSHandshakeTimeout:   api.DefaultClientTLSHandshakeTimeout,

			DialContext:        cfg.dialContext,
			DisableKeepAlives:  false,
			DisableCompression: false,
			ForceAttemptHTTP2:  true,
		},
		// by default Timeout will be 0 (meaning no timeout)
		// it can be overwritten with [WithTimeout]
		Timeout: cfg.requestTimeout,
	}
	return &client{
		socketPath:       cfg.socketPath,
		resolverClient:   resolver.NewResolverClient(c),
		engineClient:     pluginsv1connect.NewPluginManagementServiceClient(c, "http://unix"),
		versionClient:    healthv1connect.NewVersionServiceClient(c, "http://unix"),
		authorizerClient: resolver.NewAuthorizerClient(c),
	}, nil
}

func (c client) ListPlugins(ctx context.Context) ([]PluginInfo, error) {
	req := connect.NewRequest(pluginsv1.ListPluginsRequest_builder{}.Build())
	resp, err := c.engineClient.ListPlugins(ctx, req)
	if isDialError(err) {
		return nil, c.connectError(err)
	}
	if err != nil {
		return nil, err
	}
	var result []PluginInfo
	for _, item := range resp.Msg.GetPlugins() {
		name, err := api.NewName(item.GetName())
		if err != nil {
			continue
		}
		version, err := api.NewVersion(item.GetVersion())
		if err != nil {
			continue
		}
		info := PluginInfo{
			Name:          name,
			Version:       version,
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
	_, err := c.engineClient.EnablePlugin(ctx, connect.NewRequest(r))
	if isDialError(err) {
		return c.connectError(err)
	}
	return err
}

func (c client) DisablePlugin(ctx context.Context, name string) error {
	r := pluginsv1.DisablePluginRequest_builder{}.Build()
	r.SetName(name)
	_, err := c.engineClient.DisablePlugin(ctx, connect.NewRequest(r))
	if isDialError(err) {
		return c.connectError(err)
	}
	return err
}

type PluginInfo struct {
	Name                api.Name
	Version             api.Version
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
