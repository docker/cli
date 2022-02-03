package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

func TestBuild(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-buildx", `#!/bin/sh
echo '{"SchemaVersion":"0.1.0","Vendor":"Docker Inc.","Version":"v0.6.3","ShortDescription":"Build with BuildKit"}'`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	var b bytes.Buffer
	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(&b))
	assert.NilError(t, err)
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	args, os.Args, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{builderDefaultPlugin, "build", "."}, args)
}

func TestBuildkitDisabled(t *testing.T) {
	defer env.Patch(t, "DOCKER_BUILDKIT", "0")()
	var b bytes.Buffer

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(&b))
	assert.NilError(t, err)

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	args, os.Args, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"build", "."}, args)
}
