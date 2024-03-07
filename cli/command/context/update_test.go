package context

import (
	"context"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestUpdateDescriptionOnly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cli := makeFakeCli(t)
	err := RunCreate(ctx, cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
	cli.OutBuffer().Reset()
	cli.ErrBuffer().Reset()

	assert.NilError(t, RunUpdate(ctx, cli, &UpdateOptions{
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
	createTestContext(t, cli, "test")

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	assert.NilError(t, RunUpdate(ctx, cli, &UpdateOptions{
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
	assert.Check(t, cmp.Contains(c.Endpoints, docker.DockerEndpoint))
	assert.Equal(t, c.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta).Host, "tcp://some-host")
}

func TestUpdateInvalidDockerHost(t *testing.T) {
	cli := makeFakeCli(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err := RunCreate(ctx, cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)

	err = RunUpdate(ctx, cli, &UpdateOptions{
		Name: "test",
		Docker: map[string]string{
			keyHost: "some///invalid/host",
		},
	})
	assert.ErrorContains(t, err, "unable to parse docker host")
}
