package stack

import (
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/e2e/internal/stack"
	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/cli/internal/test/environment/kubernetes"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

var pollSettings = environment.DefaultPollSettings

func TestDeploy(t *testing.T) {
	configdir := fixtures.SetupConfigFile(t, kubernetes.WithOrchestrator)
	defer configdir.Remove()

	stackname := "test-stack-deploy"
	stack.DeployFullStack(t, pollSettings, stackname,
		fixtures.WithConfig(configdir.Path()),
		kubernetes.WithKubeConfig(kubeconfigPath),
	)

	defer stack.CleanupFullStack(t, pollSettings, stackname,
		fixtures.WithConfig(configdir.Path()),
		kubernetes.WithKubeConfig(kubeconfigPath),
	)

	result := icmd.RunCmd(
		environment.Shell(t, "docker stack ls"),
		fixtures.WithConfig(configdir.Path()),
		kubernetes.WithKubeConfig(kubeconfigPath),
	)
	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(), "stack-deploy-ls.golden")

	result = icmd.RunCmd(
		environment.Shell(t, "docker stack rm %s", stackname),
		fixtures.WithConfig(configdir.Path()),
		kubernetes.WithKubeConfig(kubeconfigPath),
	)
	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(), "stack-remove-success.golden")
}
