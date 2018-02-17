package checkpoint

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCheckpointListErrors(t *testing.T) {
	testCases := []struct {
		args               []string
		checkpointListFunc func(container string, options types.CheckpointListOptions) ([]types.Checkpoint, error)
		expectedError      string
	}{
		{
			args:          []string{},
			expectedError: "requires exactly 1 argument",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires exactly 1 argument",
		},
		{
			args: []string{"foo"},
			checkpointListFunc: func(container string, options types.CheckpointListOptions) ([]types.Checkpoint, error) {
				return []types.Checkpoint{}, errors.Errorf("error getting checkpoints for container foo")
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
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointListWithOptions(t *testing.T) {
	var containerID, checkpointDir string
	cli := test.NewFakeCli(&fakeClient{
		checkpointListFunc: func(container string, options types.CheckpointListOptions) ([]types.Checkpoint, error) {
			containerID = container
			checkpointDir = options.CheckpointDir
			return []types.Checkpoint{
				{Name: "checkpoint-foo"},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{"container-foo"})
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "container-foo", containerID)
	assert.Equal(t, "/dir/foo", checkpointDir)
	golden.Assert(t, cli.OutBuffer().String(), "checkpoint-list-with-options.golden")
}
