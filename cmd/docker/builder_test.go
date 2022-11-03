package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

var pluginFilename = "docker-buildx"

func TestBuildWithBuildx(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh
echo '{"SchemaVersion":"0.1.0","Vendor":"Docker Inc.","Version":"v0.6.3","ShortDescription":"Build with BuildKit"}'`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	var b bytes.Buffer

	t.Setenv("DOCKER_CONTEXT", "default")
	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(&b))
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
	assert.DeepEqual(t, []string{builderDefaultPlugin, "build", "."}, args)
	assert.DeepEqual(t, []string{"BUILDX_BUILDER=default"}, envs)
}

func TestBuildWithBuildxAndBuilder(t *testing.T) {
	t.Setenv("BUILDX_BUILDER", "mybuilder")

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh
echo '{"SchemaVersion":"0.1.0","Vendor":"Docker Inc.","Version":"v0.6.3","ShortDescription":"Build with BuildKit"}'`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	var b bytes.Buffer

	t.Setenv("DOCKER_CONTEXT", "default")
	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(&b))
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
	assert.DeepEqual(t, []string{builderDefaultPlugin, "build", "."}, args)
	assert.Check(t, len(envs) == 0)
}

func TestBuildkitDisabled(t *testing.T) {
	t.Setenv("DOCKER_BUILDKIT", "0")

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(b))
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

	output.Assert(t, b.String(), map[int]func(string) error{
		0: output.Prefix("failed to fetch metadata:"),
		2: output.Suffix("DEPRECATED: The legacy builder is deprecated and will be removed in a future release."),
	})
}

func TestBuilderBrokenEnforced(t *testing.T) {
	t.Setenv("DOCKER_BUILDKIT", "1")

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(pluginFilename, `#!/bin/sh exit 1`, fs.WithMode(0777)),
	)
	defer dir.Remove()

	b := bytes.NewBuffer(nil)

	dockerCli, err := command.NewDockerCli(command.WithInputStream(discard), command.WithCombinedStreams(b))
	assert.NilError(t, err)
	assert.NilError(t, dockerCli.Initialize(flags.NewClientOptions()))
	dockerCli.ConfigFile().CLIPluginsExtraDirs = []string{dir.Path()}

	tcmd := newDockerCommand(dockerCli)
	tcmd.SetArgs([]string{"build", "."})

	cmd, args, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)

	var envs []string
	args, os.Args, envs, err = processBuilder(dockerCli, cmd, args, os.Args)
	assert.DeepEqual(t, []string{"build", "."}, args)
	assert.Check(t, len(envs) == 0)

	output.Assert(t, err.Error(), map[int]func(string) error{
		0: output.Prefix("failed to fetch metadata:"),
		2: output.Suffix("ERROR: BuildKit is enabled but the buildx component is missing or broken."),
	})
}

func TestHasBuilderName(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		envs     []string
		expected bool
	}{
		{
			name:     "no args",
			args:     []string{"docker", "build", "."},
			envs:     []string{"FOO=bar"},
			expected: false,
		},
		{
			name:     "env var",
			args:     []string{"docker", "build", "."},
			envs:     []string{"BUILDX_BUILDER=foo"},
			expected: true,
		},
		{
			name:     "empty env var",
			args:     []string{"docker", "build", "."},
			envs:     []string{"BUILDX_BUILDER="},
			expected: false,
		},
		{
			name:     "flag",
			args:     []string{"docker", "build", "--builder", "foo", "."},
			envs:     []string{"FOO=bar"},
			expected: true,
		},
		{
			name:     "both",
			args:     []string{"docker", "build", "--builder", "foo", "."},
			envs:     []string{"BUILDX_BUILDER=foo"},
			expected: true,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasBuilderName(tt.args, tt.envs))
		})
	}
}
