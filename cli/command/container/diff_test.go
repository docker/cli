package container

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunDiff(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerDiffFunc: func(ctx context.Context, containerID string) (client.ContainerDiffResult, error) {
			return client.ContainerDiffResult{
				Changes: []container.FilesystemChange{
					{
						Kind: container.ChangeModify,
						Path: "/path/to/file0",
					},
					{
						Kind: container.ChangeAdd,
						Path: "/path/to/file1",
					},
					{
						Kind: container.ChangeDelete,
						Path: "/path/to/file2",
					},
				},
			}, nil
		},
	})

	cmd := newDiffCommand(cli)
	cmd.SetOut(io.Discard)

	cmd.SetArgs([]string{"container-id"})

	err := cmd.Execute()
	assert.NilError(t, err)

	diff := strings.SplitN(cli.OutBuffer().String(), "\n", 3)
	assert.Assert(t, is.Len(diff, 3))

	file0 := strings.TrimSpace(diff[0])
	file1 := strings.TrimSpace(diff[1])
	file2 := strings.TrimSpace(diff[2])

	assert.Check(t, is.Equal(file0, "C /path/to/file0"))
	assert.Check(t, is.Equal(file1, "A /path/to/file1"))
	assert.Check(t, is.Equal(file2, "D /path/to/file2"))
}

func TestRunDiffClientError(t *testing.T) {
	clientError := errors.New("client error")

	cli := test.NewFakeCli(&fakeClient{
		containerDiffFunc: func(ctx context.Context, containerID string) (client.ContainerDiffResult, error) {
			return client.ContainerDiffResult{}, clientError
		},
	})

	cmd := newDiffCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	cmd.SetArgs([]string{"container-id"})

	err := cmd.Execute()
	assert.ErrorIs(t, err, clientError)
}
