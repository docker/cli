package manager

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestPluginResourceAttributesEnvvar(t *testing.T) {
	cmd := &cobra.Command{
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "docker",
		},
	}

	// Ensure basic usage is fine.
	env := appendPluginResourceAttributesEnvvar(nil, cmd, Plugin{Name: "compose"})
	assert.DeepEqual(t, []string{"OTEL_RESOURCE_ATTRIBUTES=docker.cli.cobra.command_path=docker%20compose"}, env)

	// Add a user-based environment variable to OTEL_RESOURCE_ATTRIBUTES.
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "a.b.c=foo")

	env = appendPluginResourceAttributesEnvvar(nil, cmd, Plugin{Name: "compose"})
	assert.DeepEqual(t, []string{"OTEL_RESOURCE_ATTRIBUTES=a.b.c=foo,docker.cli.cobra.command_path=docker%20compose"}, env)
}

func TestPluginStubRunEReturnsParseError(t *testing.T) {
	cmd, err := preparePluginStubCommand(t)
	assert.NilError(t, err)

	err = cmd.RunE(cmd, []string{"--definitely-not-a-real-flag"})
	assert.ErrorContains(t, err, "unknown flag: --definitely-not-a-real-flag")
}

func TestPluginStubCompletionRestoresOSArgs(t *testing.T) {
	cmd, err := preparePluginStubCommand(t)
	assert.NilError(t, err)

	savedArgs := os.Args
	t.Cleanup(func() { os.Args = savedArgs })

	originalArgs := []string{"docker", "image", "ls"}
	os.Args = append([]string(nil), originalArgs...)

	_, directive := cmd.ValidArgsFunction(cmd, []string{"--all"}, "alp")
	assert.Equal(t, directive, cobra.ShellCompDirectiveError)
	assert.DeepEqual(t, os.Args, originalArgs)
}

func preparePluginStubCommand(t *testing.T) (*cobra.Command, error) {
	t.Helper()
	pluginCommandStubsOnce = sync.Once{}

	tmpDir := t.TempDir()
	const cliPlugin = `#!/bin/sh
printf '%s' '{"SchemaVersion":"0.1.0"}'
`
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-testplugin"), []byte(cliPlugin), 0o777); err != nil {
		return nil, err
	}

	cli := test.NewFakeCli(nil)
	cli.ConfigFile().CLIPluginsExtraDirs = []string{tmpDir}

	root := &cobra.Command{Use: "docker"}
	root.PersistentFlags().Bool("debug", false, "")

	if err := AddPluginCommandStubs(cli, root); err != nil {
		return nil, err
	}

	cmd, _, err := root.Find([]string{"testplugin"})
	if err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, os.ErrNotExist
	}
	return cmd, nil
}
