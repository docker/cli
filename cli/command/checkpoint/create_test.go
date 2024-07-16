package checkpoint

import (
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/checkpoint"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCheckpointCreateErrors(t *testing.T) {
	testCases := []struct {
		args                 []string
		checkpointCreateFunc func(container string, options checkpoint.CreateOptions) error
		expectedError        string
	}{
		{
			args:          []string{"too-few-arguments"},
			expectedError: "requires 2 arguments",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires 2 arguments",
		},
		{
			args: []string{"foo", "bar"},
			checkpointCreateFunc: func(container string, options checkpoint.CreateOptions) error {
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCheckpointCreateWithOptions(t *testing.T) {
	var containerID, checkpointID, checkpointDir string
	var exit bool
	cli := test.NewFakeCli(&fakeClient{
		checkpointCreateFunc: func(container string, options checkpoint.CreateOptions) error {
			containerID = container
			checkpointID = options.CheckpointID
			checkpointDir = options.CheckpointDir
			exit = options.Exit
			return nil
		},
	})
	cmd := newCreateCommand(cli)
	cp := "checkpoint-bar"
	cmd.SetArgs([]string{"container-foo", cp})
	cmd.Flags().Set("leave-running", "true")
	cmd.Flags().Set("checkpoint-dir", "/dir/foo")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("container-foo", containerID))
	assert.Check(t, is.Equal(cp, checkpointID))
	assert.Check(t, is.Equal("/dir/foo", checkpointDir))
	assert.Check(t, is.Equal(false, exit))
	assert.Check(t, is.Equal(cp, strings.TrimSpace(cli.OutBuffer().String())))
}
