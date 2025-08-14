package context

import (
	"testing"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/context/docker"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestUpdateDescriptionOnly(t *testing.T) {
	fakeCli := makeFakeCli(t)
	err := RunCreate(fakeCli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
	fakeCli.OutBuffer().Reset()
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, RunUpdate(fakeCli, &UpdateOptions{
		Name:        "test",
		Description: "description",
	}))
	c, err := fakeCli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	dc, err := cli.GetDockerContext(c)
	assert.NilError(t, err)
	assert.Equal(t, dc.Description, "description")

	assert.Equal(t, "test\n", fakeCli.OutBuffer().String())
	assert.Equal(t, "Successfully updated context \"test\"\n", fakeCli.ErrBuffer().String())
}

func TestUpdateDockerOnly(t *testing.T) {
	fakeCli := makeFakeCli(t)
	createTestContext(t, fakeCli, "test", nil)
	assert.NilError(t, RunUpdate(fakeCli, &UpdateOptions{
		Name: "test",
		Docker: map[string]string{
			keyHost: "tcp://some-host",
		},
	}))
	c, err := fakeCli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	dc, err := cli.GetDockerContext(c)
	assert.NilError(t, err)
	assert.Equal(t, dc.Description, "description of test")
	assert.Check(t, is.Contains(c.Endpoints, docker.DockerEndpoint))
	assert.Equal(t, c.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta).Host, "tcp://some-host")
}

func TestUpdateInvalidDockerHost(t *testing.T) {
	cli := makeFakeCli(t)
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
