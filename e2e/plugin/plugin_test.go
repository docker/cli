package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/e2e/testutils"
	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/skip"
)

const registryPrefix = "registry:5000"

func TestCreatePushPull(t *testing.T) {
	skip.If(t, environment.SkipPluginTests())

	const pluginName = registryPrefix + "/my-plugin"

	// TODO(thaJeztah): probably should use a config without the content trust bits.
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	pluginDir := testutils.SetupPlugin(t, ctx)

	icmd.RunCommand("docker", "plugin", "create", pluginName, pluginDir).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Command("docker", "plugin", "push", pluginName),
		fixtures.WithConfig(dir.Path()),
	)
	result.Assert(t, icmd.Expected{
		Out: fmt.Sprintf("The push refers to repository [%s]", pluginName),
	})

	icmd.RunCommand("docker", "plugin", "rm", "-f", pluginName).Assert(t, icmd.Success)

	result = icmd.RunCmd(icmd.Command("docker", "plugin", "install", "--grant-all-permissions", pluginName),
		fixtures.WithConfig(dir.Path()),
	)
	result.Assert(t, icmd.Expected{
		Out: "Installed plugin " + pluginName,
	})
}

func TestInstall(t *testing.T) {
	skip.If(t, environment.SkipPluginTests())

	const pluginName = "tiborvass/sample-volume-plugin:latest"
	result := icmd.RunCmd(icmd.Command("docker", "plugin", "install", "--grant-all-permissions", pluginName))
	result.Assert(t, icmd.Expected{
		Out: "Installed plugin " + pluginName,
	})
}
