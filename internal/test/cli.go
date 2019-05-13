package test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cli/trust"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/client"
	notaryclient "github.com/theupdateframework/notary/client"
)

// NotaryClientFuncType defines a function that returns a fake notary client
type NotaryClientFuncType func(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)
type clientInfoFuncType func() command.ClientInfo
type containerizedEngineFuncType func(string) (clitypes.ContainerizedClient, error)

// FakeCli emulates the default DockerCli
type FakeCli struct {
	command.DockerCli
	client                        client.APIClient
	configfile                    *configfile.ConfigFile
	out                           *streams.Out
	outBuffer                     *bytes.Buffer
	err                           *bytes.Buffer
	in                            *streams.In
	server                        command.ServerInfo
	clientInfoFunc                clientInfoFuncType
	notaryClientFunc              NotaryClientFuncType
	manifestStore                 manifeststore.Store
	registryClient                registryclient.RegistryClient
	contentTrust                  bool
	containerizedEngineClientFunc containerizedEngineFuncType
	contextStore                  store.Store
	currentContext                string
	dockerEndpoint                docker.Endpoint
}

// NewFakeCli returns a fake for the command.Cli interface
func NewFakeCli(client client.APIClient, opts ...func(*FakeCli)) *FakeCli {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	c := &FakeCli{
		client:    client,
		out:       streams.NewOut(outBuffer),
		outBuffer: outBuffer,
		err:       errBuffer,
		in:        streams.NewIn(ioutil.NopCloser(strings.NewReader(""))),
		// Use an empty string for filename so that tests don't create configfiles
		// Set cli.ConfigFile().Filename to a tempfile to support Save.
		configfile: configfile.New(""),
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
func (c *FakeCli) SetErr(err *bytes.Buffer) {
	c.err = err
}

// SetOut sets the stdout stream for the cli to the specified io.Writer
func (c *FakeCli) SetOut(out *streams.Out) {
	c.out = out
}

// SetConfigFile sets the "fake" config file
func (c *FakeCli) SetConfigFile(configfile *configfile.ConfigFile) {
	c.configfile = configfile
}

// SetContextStore sets the "fake" context store
func (c *FakeCli) SetContextStore(store store.Store) {
	c.contextStore = store
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

// Out returns the output stream (stdout) the cli should write on
func (c *FakeCli) Out() *streams.Out {
	return c.out
}

// Err returns the output stream (stderr) the cli should write on
func (c *FakeCli) Err() io.Writer {
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

// ClientInfo returns client information
func (c *FakeCli) ClientInfo() command.ClientInfo {
	if c.clientInfoFunc != nil {
		return c.clientInfoFunc()
	}
	return c.DockerCli.ClientInfo()
}

// SetClientInfo sets the internal getter for retrieving a ClientInfo
func (c *FakeCli) SetClientInfo(clientInfoFunc clientInfoFuncType) {
	c.clientInfoFunc = clientInfoFunc
}

// OutBuffer returns the stdout buffer
func (c *FakeCli) OutBuffer() *bytes.Buffer {
	return c.outBuffer
}

// ErrBuffer Buffer returns the stderr buffer
func (c *FakeCli) ErrBuffer() *bytes.Buffer {
	return c.err
}

// ResetOutputBuffers resets the .OutBuffer() and.ErrBuffer() back to empty
func (c *FakeCli) ResetOutputBuffers() {
	c.outBuffer.Reset()
	c.err.Reset()
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
	return nil, fmt.Errorf("no notary client available unless defined")
}

// ManifestStore returns a fake store used for testing
func (c *FakeCli) ManifestStore() manifeststore.Store {
	return c.manifestStore
}

// RegistryClient returns a fake client for testing
func (c *FakeCli) RegistryClient(insecure bool) registryclient.RegistryClient {
	return c.registryClient
}

// SetManifestStore on the fake cli
func (c *FakeCli) SetManifestStore(store manifeststore.Store) {
	c.manifestStore = store
}

// SetRegistryClient on the fake cli
func (c *FakeCli) SetRegistryClient(client registryclient.RegistryClient) {
	c.registryClient = client
}

// ContentTrustEnabled on the fake cli
func (c *FakeCli) ContentTrustEnabled() bool {
	return c.contentTrust
}

// EnableContentTrust on the fake cli
func EnableContentTrust(c *FakeCli) {
	c.contentTrust = true
}

// NewContainerizedEngineClient returns a containerized engine client
func (c *FakeCli) NewContainerizedEngineClient(sockPath string) (clitypes.ContainerizedClient, error) {
	if c.containerizedEngineClientFunc != nil {
		return c.containerizedEngineClientFunc(sockPath)
	}
	return nil, fmt.Errorf("no containerized engine client available unless defined")
}

// SetContainerizedEngineClient on the fake cli
func (c *FakeCli) SetContainerizedEngineClient(containerizedEngineClientFunc containerizedEngineFuncType) {
	c.containerizedEngineClientFunc = containerizedEngineClientFunc
}

// StackOrchestrator return the selected stack orchestrator
func (c *FakeCli) StackOrchestrator(flagValue string) (command.Orchestrator, error) {
	configOrchestrator := ""
	if c.configfile != nil {
		configOrchestrator = c.configfile.StackOrchestrator
	}
	ctxOrchestrator := ""
	if c.currentContext != "" && c.contextStore != nil {
		meta, err := c.contextStore.GetMetadata(c.currentContext)
		if err != nil {
			return "", err
		}
		context, err := command.GetDockerContext(meta)
		if err != nil {
			return "", err
		}
		ctxOrchestrator = string(context.StackOrchestrator)
	}
	return command.GetStackOrchestrator(flagValue, ctxOrchestrator, configOrchestrator, c.err)
}
