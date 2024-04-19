package checkpoint

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/checkpoint"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCheckpointRemoveErrors(t *testing.T) {
	testCases := []struct {
		args                 []string
		checkpointDeleteFunc func(container string, options checkpoint.DeleteOptions) error
		expectedError        string
	}{
		{
			args:          []string{"too-few-arguments"},
			expectedError: "requires exactly 2 arguments",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires exactly 2 arguments",
		},
		{
			args: []string{"foo", "bar"},
			checkpointDeleteFunc: func(container string, options checkpoint.DeleteOptions) error {
				return errors.Errorf("error deleting checkpoint")
			},
			expectedError: "error deleting checkpoint",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			checkpointDeleteFunc: tc.checkpointDeleteFunc,
		})
		cmd := newRemoveCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointRemoveWithOptions(t *testing.T) {
	var containerID, checkpointID, checkpointDir string
	cli := test.NewFakeCli(&fakeClient{
		checkpointDeleteFunc: func(container string, options checkpoint.DeleteOptions) error {
			containerID = container
			checkpointID = options.CheckpointID
			checkpointDir = options.CheckpointDir
			return nil
		},
	})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"container-foo", "checkpoint-bar"})
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("container-foo", containerID))
	assert.Check(t, is.Equal("checkpoint-bar", checkpointID))
	assert.Check(t, is.Equal("/dir/foo", checkpointDir))
}
