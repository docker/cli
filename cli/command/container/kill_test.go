package container

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunKill(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerKillFunc: func(ctx context.Context, container string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
			assert.Assert(t, is.Equal(options.Signal, "STOP"))
			return client.ContainerKillResult{}, nil
		},
	})

	cmd := newKillCommand(cli)
	cmd.SetOut(io.Discard)

	cmd.SetArgs([]string{
		"--signal", "STOP",
		"container-id-1",
		"container-id-2",
	})
	err := cmd.Execute()
	assert.NilError(t, err)

	containerIDs := strings.SplitN(cli.OutBuffer().String(), "\n", 2)
	assert.Assert(t, is.Len(containerIDs, 2))

	containerID1 := strings.TrimSpace(containerIDs[0])
	containerID2 := strings.TrimSpace(containerIDs[1])

	assert.Check(t, is.Equal(containerID1, "container-id-1"))
	assert.Check(t, is.Equal(containerID2, "container-id-2"))
}

func TestRunKillClientError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerKillFunc: func(ctx context.Context, container string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
			return client.ContainerKillResult{}, fmt.Errorf("client error for container %s", container)
		},
	})

	cmd := newKillCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	cmd.SetArgs([]string{"container-id-1", "container-id-2"})
	err := cmd.Execute()

	errs := strings.SplitN(err.Error(), "\n", 2)
	assert.Assert(t, is.Len(errs, 2))

	errContainerID1 := errs[0]
	errContainerID2 := errs[1]

	assert.Assert(t, is.Equal(errContainerID1, "client error for container container-id-1"))
	assert.Assert(t, is.Equal(errContainerID2, "client error for container container-id-2"))
}
