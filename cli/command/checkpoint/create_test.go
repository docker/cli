package checkpoint

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestCheckpointCreateErrors(t *testing.T) {
	testCases := []struct {
		args                 []string
		checkpointCreateFunc func(container string, options types.CheckpointCreateOptions) error
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
			checkpointCreateFunc: func(container string, options types.CheckpointCreateOptions) error {
				return errors.Errorf("error creating checkpoint for container foo")
			},
			expectedError: "error creating checkpoint for container foo",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			checkpointCreateFunc: tc.checkpointCreateFunc,
		})
		cmd := newCreateCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointCreateWithOptions(t *testing.T) {
	var containerID, checkpointID, checkpointDir string
	var exit bool
	cli := test.NewFakeCli(&fakeClient{
		checkpointCreateFunc: func(container string, options types.CheckpointCreateOptions) error {
			containerID = container
			checkpointID = options.CheckpointID
			checkpointDir = options.CheckpointDir
			exit = options.Exit
			return nil
		},
	})
	cmd := newCreateCommand(cli)
	checkpoint := "checkpoint-bar"
	cmd.SetArgs([]string{"container-foo", checkpoint})
	cmd.Flags().Set("leave-running", "true")
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("container-foo", containerID))
	assert.Check(t, is.Equal(checkpoint, checkpointID))
	assert.Check(t, is.Equal("/dir/foo", checkpointDir))
	assert.Check(t, is.Equal(false, exit))
	assert.Check(t, is.Equal(checkpoint, strings.TrimSpace(cli.OutBuffer().String())))
}
