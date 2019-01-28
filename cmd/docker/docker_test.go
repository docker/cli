package main

import (
	"bytes"
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

	cmd := newDockerCommand(&command.DockerCli{})
	cmd.Flags().Set("debug", "true")

	err := cmd.PersistentPreRunE(cmd, []string{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("1", os.Getenv("DEBUG")))
	assert.Check(t, is.Equal(logrus.DebugLevel, logrus.GetLevel()))
}

var discard = ioutil.NopCloser(bytes.NewBuffer(nil))

func TestExitStatusForInvalidSubcommandWithHelpFlag(t *testing.T) {
	cli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(ioutil.Discard))
	assert.NilError(t, err)
	cmd := newDockerCommand(cli)
	cmd.SetArgs([]string{"help", "invalid"})
	err = cmd.Execute()
	assert.Error(t, err, "unknown help topic: invalid")
}

func TestExitStatusForInvalidSubcommand(t *testing.T) {
	cli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(ioutil.Discard))
	assert.NilError(t, err)
	cmd := newDockerCommand(cli)
	cmd.SetArgs([]string{"invalid"})
	err = cmd.Execute()
	assert.Check(t, is.ErrorContains(err, "docker: 'invalid' is not a docker command."))
}

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	cli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(&b))
	assert.NilError(t, err)
	cmd := newDockerCommand(cli)
	cmd.SetArgs([]string{"--version"})
	err = cmd.Execute()
	assert.NilError(t, err)
	assert.Check(t, is.Contains(b.String(), "Docker version"))
}
