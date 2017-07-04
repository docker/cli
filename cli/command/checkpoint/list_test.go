package checkpoint

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
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
		}, &bytes.Buffer{})
		cmd := newListCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointListWithOptions(t *testing.T) {
	var containerID, checkpointDir string
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		checkpointListFunc: func(container string, options types.CheckpointListOptions) ([]types.Checkpoint, error) {
			containerID = container
			checkpointDir = options.CheckpointDir
			return []types.Checkpoint{
				{Name: "checkpoint-foo"},
			}, nil
		},
	}, buf)
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{"container-foo"})
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "container-foo", containerID)
	assert.Equal(t, "/dir/foo", checkpointDir)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "checkpoint-list-with-options.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
