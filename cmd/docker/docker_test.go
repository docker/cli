package main

import (
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

func TestExitStatusForInvalidSubcommandWithHelpFlag(t *testing.T) {
	discard := ioutil.Discard
	cmd := newDockerCommand(command.NewDockerCli(os.Stdin, discard, discard, false, nil))
	cmd.SetArgs([]string{"help", "invalid"})
	err := cmd.Execute()
	assert.Error(t, err, "unknown help topic: invalid")
}
