package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestUpdateDescriptionOnly(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	err := RunCreate(cli, &CreateOptions{
		Name:                     "test",
		DefaultStackOrchestrator: "swarm",
		Docker:                   map[string]string{},
	})
	assert.NilError(t, err)
	cli.OutBuffer().Reset()
	cli.ErrBuffer().Reset()
	assert.NilError(t, RunUpdate(cli, &UpdateOptions{
		Name:        "test",
		Description: "description",
	}))
	c, err := cli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	dc, err := command.GetDockerContext(c)
	assert.NilError(t, err)
	assert.Equal(t, dc.StackOrchestrator, command.OrchestratorSwarm)
	assert.Equal(t, dc.Description, "description")

	assert.Equal(t, "test\n", cli.OutBuffer().String())
	assert.Equal(t, "Successfully updated context \"test\"\n", cli.ErrBuffer().String())
}

func TestUpdateDockerOnly(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "test", "swarm")
	assert.NilError(t, RunUpdate(cli, &UpdateOptions{
		Name: "test",
		Docker: map[string]string{
			keyHost: "tcp://some-host",
		},
	}))
	c, err := cli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	dc, err := command.GetDockerContext(c)
	assert.NilError(t, err)
	assert.Equal(t, dc.StackOrchestrator, command.OrchestratorSwarm)
	assert.Equal(t, dc.Description, "description of test")
	assert.Check(t, cmp.Contains(c.Endpoints, kubernetes.KubernetesEndpoint))
	assert.Check(t, cmp.Contains(c.Endpoints, docker.DockerEndpoint))
	assert.Equal(t, c.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta).Host, "tcp://some-host")
}

func TestUpdateStackOrchestratorStrategy(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	err := RunCreate(cli, &CreateOptions{
		Name:                     "test",
		DefaultStackOrchestrator: "swarm",
		Docker:                   map[string]string{},
	})
	assert.NilError(t, err)
	err = RunUpdate(cli, &UpdateOptions{
		Name:                     "test",
		DefaultStackOrchestrator: "kubernetes",
	})
	assert.ErrorContains(t, err, `cannot specify orchestrator "kubernetes" without configuring a Kubernetes endpoint`)
}

func TestUpdateStackOrchestratorStrategyRemoveKubeEndpoint(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "test", "kubernetes")
	err := RunUpdate(cli, &UpdateOptions{
		Name:       "test",
		Kubernetes: map[string]string{},
	})
	assert.ErrorContains(t, err, `cannot specify orchestrator "kubernetes" without configuring a Kubernetes endpoint`)
}

func TestUpdateInvalidDockerHost(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
	err = RunUpdate(cli, &UpdateOptions{
		Name: "test",
		Docker: map[string]string{
			keyHost: "some///invalid/host",
		},
	})
	assert.ErrorContains(t, err, "unable to parse docker host")
}
