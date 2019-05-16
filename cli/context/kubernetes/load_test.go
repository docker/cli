package kubernetes

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	cliflags "github.com/docker/cli/cli/flags"
	"gotest.tools/assert"
	"gotest.tools/env"
)

func TestDefaultContextInitializer(t *testing.T) {
	cli, err := command.NewDockerCli()
	assert.NilError(t, err)
	defer env.Patch(t, "KUBECONFIG", "./testdata/test-kubeconfig")()
	configFile := &configfile.ConfigFile{
		StackOrchestrator: "all",
	}
	ctx, err := command.ResolveDefaultContext(&cliflags.CommonOptions{}, configFile, command.DefaultContextStoreConfig(), cli.Err())
	assert.NilError(t, err)
	assert.Equal(t, "default", ctx.Meta.Name)
	assert.Equal(t, command.OrchestratorAll, ctx.Meta.Metadata.(command.DockerContext).StackOrchestrator)
	assert.DeepEqual(t, "zoinx", ctx.Meta.Endpoints[KubernetesEndpoint].(EndpointMeta).DefaultNamespace)
}
