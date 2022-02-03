package main

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

var pluginFilename = "docker-buildx"

func init() {
	if runtime.GOOS == "windows" {
		pluginFilename = pluginFilename + ".exe"
	}
}

func TestBuildWithBuilder(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh
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

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(b))
	assert.NilError(t, err)
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	args, os.Args, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"build", "."}, args)

	output.Assert(t, b.String(), map[int]func(string) error{
		0: output.Suffix("DEPRECATED: The legacy builder is deprecated and will be removed in a future release."),
	})
}

func TestBuilderBroken(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(b))
	assert.NilError(t, err)
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	args, os.Args, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"build", "."}, args)

	output.Assert(t, b.String(), map[int]func(string) error{
		0: output.Prefix("failed to fetch metadata:"),
		2: output.Suffix("DEPRECATED: The legacy builder is deprecated and will be removed in a future release."),
	})
}

func TestBuilderBrokenEnforced(t *testing.T) {
	defer env.Patch(t, "DOCKER_BUILDKIT", "1")()

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(b))
	assert.NilError(t, err)
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	args, os.Args, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.DeepEqual(t, []string{"build", "."}, args)

	output.Assert(t, err.Error(), map[int]func(string) error{
		0: output.Prefix("failed to fetch metadata:"),
		2: output.Suffix("ERROR: BuildKit is enabled but the buildx component is missing or broken."),
	})
}
