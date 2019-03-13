package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/debug"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestClientDebugEnabled(t *testing.T) {
	defer debug.Disable()

	tcmd := newDockerCommand(&command.DockerCli{})
	tcmd.SetFlag("debug", "true")
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("1", os.Getenv("DEBUG")))
	assert.Check(t, is.Equal(logrus.DebugLevel, logrus.GetLevel()))
}

var discard = ioutil.NopCloser(bytes.NewBuffer(nil))

func runCliCommand(t *testing.T, r io.ReadCloser, w io.Writer, args ...string) error {
	t.Helper()
	if r == nil {
		r = discard
	}
	if w == nil {
		w = ioutil.Discard
	}
	cli, err := command.NewDockerCli(command.WithInputStream(r), command.WithCombinedStreams(w))
	assert.NilError(t, err)
	tcmd := newDockerCommand(cli)
	tcmd.SetArgs(args)
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	return cmd.Execute()
}

func TestExitStatusForInvalidSubcommandWithHelpFlag(t *testing.T) {
	err := runCliCommand(t, nil, nil, "help", "invalid")
	assert.Error(t, err, "unknown help topic: invalid")
}

func TestExitStatusForInvalidSubcommand(t *testing.T) {
	err := runCliCommand(t, nil, nil, "invalid")
	assert.Check(t, is.ErrorContains(err, "docker: 'invalid' is not a docker command."))
}

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	err := runCliCommand(t, nil, &b, "--version")
	assert.NilError(t, err)
	assert.Check(t, is.Contains(b.String(), "Docker version"))
}
