package checkpoint

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/testutil"
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
			expectedError: "requires exactly 2 argument(s)",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires exactly 2 argument(s)",
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
		cli := test.NewFakeCliWithOutput(&fakeClient{
			checkpointDeleteFunc: tc.checkpointDeleteFunc,
		}, &bytes.Buffer{})
		cmd := newRemoveCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointRemoveWithOptions(t *testing.T) {
	var containerID, checkpointID, checkpointDir string
	cli := test.NewFakeCliWithOutput(&fakeClient{
		checkpointDeleteFunc: func(container string, options types.CheckpointDeleteOptions) error {
			containerID = container
			checkpointID = options.CheckpointID
			checkpointDir = options.CheckpointDir
			return nil
		},
	}, &bytes.Buffer{})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"container-foo", "checkpoint-bar"})
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "container-foo", containerID)
	assert.Equal(t, "checkpoint-bar", checkpointID)
	assert.Equal(t, "/dir/foo", checkpointDir)
}
