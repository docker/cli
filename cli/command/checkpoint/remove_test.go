package checkpoint

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
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
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "container-foo", containerID)
	assert.Equal(t, "checkpoint-bar", checkpointID)
	assert.Equal(t, "/dir/foo", checkpointDir)
}
