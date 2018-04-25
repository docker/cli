package test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/client"
	notaryclient "github.com/theupdateframework/notary/client"
)

// NotaryClientFuncType defines a function that returns a fake notary client
type NotaryClientFuncType func(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)
type clientInfoFuncType func() command.ClientInfo

// FakeCli emulates the default DockerCli
type FakeCli struct {
	command.DockerCli
	client           client.APIClient
	configfile       *configfile.ConfigFile
	out              *command.OutStream
	outBuffer        *bytes.Buffer
	err              *bytes.Buffer
	in               *command.InStream
	server           command.ServerInfo
	clientInfoFunc   clientInfoFuncType
	notaryClientFunc NotaryClientFuncType
	manifestStore    manifeststore.Store
	registryClient   registryclient.RegistryClient
	contentTrust     bool
}

// NewFakeCli returns a fake for the command.Cli interface
func NewFakeCli(client client.APIClient, opts ...func(*FakeCli)) *FakeCli {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	c := &FakeCli{
		client:    client,
		out:       command.NewOutStream(outBuffer),
		outBuffer: outBuffer,
		err:       errBuffer,
		in:        command.NewInStream(ioutil.NopCloser(strings.NewReader(""))),
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
func (c *FakeCli) SetIn(in *command.InStream) {
	c.in = in
}

// SetErr sets the stderr stream for the cli to the specified io.Writer
func (c *FakeCli) SetErr(err *bytes.Buffer) {
	c.err = err
}

// SetConfigFile sets the "fake" config file
func (c *FakeCli) SetConfigFile(configfile *configfile.ConfigFile) {
	c.configfile = configfile
}

// Client returns a docker API client
func (c *FakeCli) Client() client.APIClient {
	return c.client
}

// Out returns the output stream (stdout) the cli should write on
func (c *FakeCli) Out() *command.OutStream {
	return c.out
}

// Err returns the output stream (stderr) the cli should write on
func (c *FakeCli) Err() io.Writer {
	return c.err
}

// In returns the input stream the cli will use
func (c *FakeCli) In() *command.InStream {
	return c.in
}

// ConfigFile returns the cli configfile object (to get client configuration)
func (c *FakeCli) ConfigFile() *configfile.ConfigFile {
	return c.configfile
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

// OrchestratorSwarm sets a command.ClientInfo with Swarm orchestrator
func OrchestratorSwarm(c *FakeCli) {
	c.SetClientInfo(func() command.ClientInfo { return command.ClientInfo{Orchestrator: "swarm"} })
}

// OrchestratorKubernetes sets a command.ClientInfo with Kubernetes orchestrator
func OrchestratorKubernetes(c *FakeCli) {
	c.SetClientInfo(func() command.ClientInfo { return command.ClientInfo{Orchestrator: "kubernetes"} })
}

// OrchestratorAll sets a command.ClientInfo with all orchestrator
func OrchestratorAll(c *FakeCli) {
	c.SetClientInfo(func() command.ClientInfo { return command.ClientInfo{Orchestrator: "all"} })
}
