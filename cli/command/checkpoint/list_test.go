package checkpoint

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/checkpoint"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestCheckpointListErrors(t *testing.T) {
	testCases := []struct {
		args               []string
		checkpointListFunc func(container string, options checkpoint.ListOptions) ([]checkpoint.Summary, error)
		expectedError      string
	}{
		{
			args:          []string{},
			expectedError: "requires 1 argument",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires 1 argument",
		},
		{
			args: []string{"foo"},
			checkpointListFunc: func(container string, options checkpoint.ListOptions) ([]checkpoint.Summary, error) {
				return []checkpoint.Summary{}, errors.New("error getting checkpoints for container foo")
			},
			expectedError: "error getting checkpoints for container foo",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			checkpointListFunc: tc.checkpointListFunc,
		})
		cmd := newListCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointListWithOptions(t *testing.T) {
	var containerID, checkpointDir string
	cli := test.NewFakeCli(&fakeClient{
		checkpointListFunc: func(container string, options checkpoint.ListOptions) ([]checkpoint.Summary, error) {
			containerID = container
			checkpointDir = options.CheckpointDir
			return []checkpoint.Summary{
				{Name: "checkpoint-foo"},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{"container-foo"})
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("container-foo", containerID))
	assert.Check(t, is.Equal("/dir/foo", checkpointDir))
	golden.Assert(t, cli.OutBuffer().String(), "checkpoint-list-with-options.golden")
}
