package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func init() {
	pluginFilename = pluginFilename + ".exe"
}

type failingPingClient struct {
	client.Client
}

func (*failingPingClient) Ping(context.Context, client.PingOptions) (client.PingResult, error) {
	return client.PingResult{}, errors.New("daemon unavailable")
}

func TestBuildWithUnavailableDaemonUsesWindowsLegacyDefault(t *testing.T) {
	ctx := t.Context()

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0o777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(
		command.WithBaseContext(ctx),
		command.WithAPIClient(&failingPingClient{}),
		command.WithInputStream(discard),
		command.WithCombinedStreams(b),
	)
	assert.NilError(t, err)
	assert.NilError(t, dockerCli.Initialize(flags.NewClientOptions()))
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	var envs []string
	args, os.Args, envs, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"build", "."}, args)
	assert.Check(t, len(envs) == 0)
	assert.Equal(t, b.String(), "")
}
