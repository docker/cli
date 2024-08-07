package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/debug"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestClientDebugEnabled(t *testing.T) {
	defer debug.Disable()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cli, err := command.NewDockerCli(command.WithBaseContext(ctx))
	assert.NilError(t, err)
	tcmd := newDockerCommand(cli)
	tcmd.SetFlag("debug", "true")
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("1", os.Getenv("DEBUG")))
	assert.Check(t, is.Equal(logrus.DebugLevel, logrus.GetLevel()))
}

var discard = io.NopCloser(bytes.NewBuffer(nil))

func runCliCommand(t *testing.T, r io.ReadCloser, w io.Writer, args ...string) error {
	t.Helper()
	if r == nil {
		r = discard
	}
	if w == nil {
		w = io.Discard
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cli, err := command.NewDockerCli(
		command.WithBaseContext(ctx),
		command.WithInputStream(r),
		command.WithCombinedStreams(w))
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
	assert.Check(t, is.ErrorContains(err, "docker: unknown command: docker invalid"))
}

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	err := runCliCommand(t, nil, &b, "--version")
	assert.NilError(t, err)
	assert.Check(t, is.Contains(b.String(), "Docker version"))
}
