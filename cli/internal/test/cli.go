package test

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/client"
)

// FakeCli emulates the default DockerCli
type FakeCli struct {
	command.DockerCli
	client     client.APIClient
	configfile *configfile.ConfigFile
	out        *command.OutStream
	outBuffer  *bytes.Buffer
	err        *bytes.Buffer
	in         *command.InStream
	server     command.ServerInfo
}

// NewFakeCliWithOutput returns a Cli backed by the fakeCli
// Deprecated: Use NewFakeCli
func NewFakeCliWithOutput(client client.APIClient, out io.Writer) *FakeCli {
	cli := NewFakeCli(client)
	cli.out = command.NewOutStream(out)
	return cli
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
