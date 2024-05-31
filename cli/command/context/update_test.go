package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestUpdateDescriptionOnly(t *testing.T) {
	cli := makeFakeCli(t)
	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
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
	assert.Equal(t, dc.Description, "description")

	assert.Equal(t, "test\n", cli.OutBuffer().String())
	assert.Equal(t, "Successfully updated context \"test\"\n", cli.ErrBuffer().String())
}

func TestUpdateDockerOnly(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "test", nil)
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
