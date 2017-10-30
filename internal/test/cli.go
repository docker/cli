package test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/client"
	notaryclient "github.com/theupdateframework/notary/client"
)

type notaryClientFuncType func(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)

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
	notaryClientFunc notaryClientFuncType
}

// NewFakeCli returns a fake for the command.Cli interface
func NewFakeCli(client client.APIClient) *FakeCli {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	return &FakeCli{
		client:     client,
		out:        command.NewOutStream(outBuffer),
		outBuffer:  outBuffer,
		err:        errBuffer,
		in:         command.NewInStream(ioutil.NopCloser(strings.NewReader(""))),
		configfile: configfile.New("configfile"),
	}
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

// OutBuffer returns the stdout buffer
func (c *FakeCli) OutBuffer() *bytes.Buffer {
	return c.outBuffer
}

// ErrBuffer Buffer returns the stderr buffer
func (c *FakeCli) ErrBuffer() *bytes.Buffer {
	return c.err
}

// SetNotaryClient sets the internal getter for retrieving a NotaryClient
func (c *FakeCli) SetNotaryClient(notaryClientFunc notaryClientFuncType) {
	c.notaryClientFunc = notaryClientFunc
}

// NotaryClient returns an err for testing unless defined
func (c *FakeCli) NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error) {
	if c.notaryClientFunc != nil {
		return c.notaryClientFunc(imgRefAndAuth, actions)
	}
	return nil, fmt.Errorf("no notary client available unless defined")
}
