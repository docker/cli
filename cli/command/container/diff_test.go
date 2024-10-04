package container

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunDiff(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerDiffFunc: func(
			ctx context.Context,
			containerID string,
		) ([]container.FilesystemChange, error) {
			return []container.FilesystemChange{
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
			}, nil
		},
	})

	cmd := NewDiffCommand(cli)
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
		containerDiffFunc: func(
			ctx context.Context,
			containerID string,
		) ([]container.FilesystemChange, error) {
			return nil, clientError
		},
	})

	cmd := NewDiffCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	cmd.SetArgs([]string{"container-id"})

	err := cmd.Execute()
	assert.ErrorIs(t, err, clientError)
}

func TestRunDiffEmptyContainerError(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})

	cmd := NewDiffCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	containerID := ""
	cmd.SetArgs([]string{containerID})

	err := cmd.Execute()
	assert.Error(t, err, "Container name cannot be empty")
}
