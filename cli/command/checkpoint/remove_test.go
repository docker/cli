package checkpoint

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestCheckpointRemoveErrors(t *testing.T) {
	testCases := []struct {
		args                 []string
		checkpointDeleteFunc func(container string, options types.CheckpointDeleteOptions) error
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
			checkpointDeleteFunc: func(container string, options types.CheckpointDeleteOptions) error {
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
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointRemoveWithOptions(t *testing.T) {
	var containerID, checkpointID, checkpointDir string
	cli := test.NewFakeCli(&fakeClient{
		checkpointDeleteFunc: func(container string, options types.CheckpointDeleteOptions) error {
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
