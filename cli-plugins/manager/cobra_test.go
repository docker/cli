package manager

import (
	"os"
	"sync"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
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
	cmd := preparePluginStubCommand(t, `{"SchemaVersion":"0.1.0","Vendor":"e2e-testing"}`)

	err := cmd.RunE(cmd, []string{"--definitely-not-a-real-flag"})
	assert.ErrorContains(t, err, "unknown flag: --definitely-not-a-real-flag")
}

func TestPluginStubCompletionRestoresOSArgs(t *testing.T) {
	cmd := preparePluginStubCommand(t, `{"SchemaVersion":"0.1.0"}`)

	originalArgs := []string{"docker", "image", "ls"}
	os.Args = append([]string(nil), originalArgs...)

	_, directive := cmd.ValidArgsFunction(cmd, []string{"--all"}, "alp")
	assert.Equal(t, directive, cobra.ShellCompDirectiveError)
	assert.DeepEqual(t, os.Args, originalArgs)
}

func preparePluginStubCommand(t *testing.T, metadata string) *cobra.Command {
	t.Helper()
	pluginCommandStubsOnce = sync.Once{}

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-testplugin", "#!/bin/sh\nprintf '%s' '"+metadata+"'\n", fs.WithMode(0o777)),
	)
	t.Cleanup(func() { dir.Remove() })

	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(&configfile.ConfigFile{
		CLIPluginsExtraDirs: []string{dir.Path()},
	})

	root := &cobra.Command{Use: "docker"}
	root.PersistentFlags().Bool("debug", false, "")

	err := AddPluginCommandStubs(cli, root)
	assert.NilError(t, err)

	cmd, _, err := root.Find([]string{"testplugin"})
	assert.NilError(t, err)
	assert.Assert(t, cmd != nil)
	return cmd
}
