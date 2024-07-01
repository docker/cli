package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/cli/oauth"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/client"
	notaryclient "github.com/theupdateframework/notary/client"
)

// NotaryClientFuncType defines a function that returns a fake notary client
type NotaryClientFuncType func(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)

// FakeCli emulates the default DockerCli
type FakeCli struct {
	command.DockerCli
	client           client.APIClient
	configfile       *configfile.ConfigFile
	out              *streams.Out
	outBuffer        *bytes.Buffer
	err              *streams.Out
	errBuffer        *bytes.Buffer
	in               *streams.In
	server           command.ServerInfo
	notaryClientFunc NotaryClientFuncType
	manifestStore    manifeststore.Store
	registryClient   registryclient.RegistryClient
	contentTrust     bool
	contextStore     store.Store
	currentContext   string
	dockerEndpoint   docker.Endpoint
	oauthManager     *fakeOauthManager
}

// NewFakeCli returns a fake for the command.Cli interface
func NewFakeCli(apiClient client.APIClient, opts ...func(*FakeCli)) *FakeCli {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	c := &FakeCli{
		client:    apiClient,
		out:       streams.NewOut(outBuffer),
		outBuffer: outBuffer,
		err:       streams.NewOut(errBuffer),
		errBuffer: errBuffer,
		in:        streams.NewIn(io.NopCloser(strings.NewReader(""))),
		// Use an empty string for filename so that tests don't create configfiles
		// Set cli.ConfigFile().Filename to a tempfile to support Save.
		configfile:     configfile.New(""),
		currentContext: command.DefaultContextName,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetIn sets the input of the cli to the specified ReadCloser
func (c *FakeCli) SetIn(in *streams.In) {
	c.in = in
}

// SetErr sets the stderr stream for the cli to the specified io.Writer
func (c *FakeCli) SetErr(err *streams.Out) {
	c.err = err
}

// SetOut sets the stdout stream for the cli to the specified io.Writer
func (c *FakeCli) SetOut(out *streams.Out) {
	c.out = out
}

// SetConfigFile sets the "fake" config file
func (c *FakeCli) SetConfigFile(configFile *configfile.ConfigFile) {
	c.configfile = configFile
}

// SetContextStore sets the "fake" context store
func (c *FakeCli) SetContextStore(contextStore store.Store) {
	c.contextStore = contextStore
}

// SetCurrentContext sets the "fake" current context
func (c *FakeCli) SetCurrentContext(name string) {
	c.currentContext = name
}

// SetDockerEndpoint sets the "fake" docker endpoint
func (c *FakeCli) SetDockerEndpoint(ep docker.Endpoint) {
	c.dockerEndpoint = ep
}

// Client returns a docker API client
func (c *FakeCli) Client() client.APIClient {
	return c.client
}

// CurrentVersion returns the API version used by FakeCli.
func (c *FakeCli) CurrentVersion() string {
	return c.DefaultVersion()
}

// Out returns the output stream (stdout) the cli should write on
func (c *FakeCli) Out() *streams.Out {
	return c.out
}

// Err returns the output stream (stderr) the cli should write on
func (c *FakeCli) Err() *streams.Out {
	return c.err
}

// In returns the input stream the cli will use
func (c *FakeCli) In() *streams.In {
	return c.in
}

// ConfigFile returns the cli configfile object (to get client configuration)
func (c *FakeCli) ConfigFile() *configfile.ConfigFile {
	return c.configfile
}

// ContextStore returns the cli context store
func (c *FakeCli) ContextStore() store.Store {
	return c.contextStore
}

// CurrentContext returns the cli context
func (c *FakeCli) CurrentContext() string {
	return c.currentContext
}

// DockerEndpoint returns the current DockerEndpoint
func (c *FakeCli) DockerEndpoint() docker.Endpoint {
	return c.dockerEndpoint
}

// ServerInfo returns API server information for the server used by this client
func (c *FakeCli) ServerInfo() command.ServerInfo {
	return c.server
}

// OutBuffer returns the stdout buffer
func (c *FakeCli) OutBuffer() *bytes.Buffer {
	return c.outBuffer
}

// ErrBuffer Buffer returns the stderr buffer
func (c *FakeCli) ErrBuffer() *bytes.Buffer {
	return c.errBuffer
}

// ResetOutputBuffers resets the .OutBuffer() and.ErrBuffer() back to empty
func (c *FakeCli) ResetOutputBuffers() {
	c.outBuffer.Reset()
	c.errBuffer.Reset()
}

// SetNotaryClient sets the internal getter for retrieving a NotaryClient
func (c *FakeCli) SetNotaryClient(notaryClientFunc NotaryClientFuncType) {
	c.notaryClientFunc = notaryClientFunc
}

// NotaryClient returns an err for testing unless defined
func (c *FakeCli) NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error) {
	if c.notaryClientFunc != nil {
		return c.notaryClientFunc(imgRefAndAuth, actions)
	}
	return nil, errors.New("no notary client available unless defined")
}

// ManifestStore returns a fake store used for testing
func (c *FakeCli) ManifestStore() manifeststore.Store {
	return c.manifestStore
}

// RegistryClient returns a fake client for testing
func (c *FakeCli) RegistryClient(bool) registryclient.RegistryClient {
	return c.registryClient
}

// SetManifestStore on the fake cli
func (c *FakeCli) SetManifestStore(manifestStore manifeststore.Store) {
	c.manifestStore = manifestStore
}

// SetRegistryClient on the fake cli
func (c *FakeCli) SetRegistryClient(registryClient registryclient.RegistryClient) {
	c.registryClient = registryClient
}

// ContentTrustEnabled on the fake cli
func (c *FakeCli) ContentTrustEnabled() bool {
	return c.contentTrust
}

// EnableContentTrust on the fake cli
func EnableContentTrust(c *FakeCli) {
	c.contentTrust = true
}

// BuildKitEnabled on the fake cli
func (c *FakeCli) BuildKitEnabled() (bool, error) {
	return true, nil
}

// OAuthManager on the fake cli
func (c *FakeCli) OAuthManager() oauth.Manager {
	return c.oauthManager
}

type fakeOauthManager struct{}

func (f *fakeOauthManager) LoginDevice(ctx context.Context, w io.Writer) (res oauth.TokenResult, err error) {
	return res, nil
}

func (f *fakeOauthManager) Logout() error {
	return nil
}

func (f *fakeOauthManager) RefreshToken() (res oauth.TokenResult, err error) {
	return res, nil
}
