package checkpoint

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/checkpoint"
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
				return errors.New("error creating checkpoint for container foo")
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
	const (
		containerName  = "container-foo"
		checkpointName = "checkpoint-bar"
		checkpointDir  = "/dir/foo"
	)

	for _, tc := range []bool{true, false} {
		leaveRunning := strconv.FormatBool(tc)
		t.Run("leave-running="+leaveRunning, func(t *testing.T) {
			var actualContainerName string
			var actualOptions checkpoint.CreateOptions
			cli := test.NewFakeCli(&fakeClient{
				checkpointCreateFunc: func(container string, options checkpoint.CreateOptions) error {
					actualContainerName = container
					actualOptions = options
					return nil
				},
			})
			cmd := newCreateCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{containerName, checkpointName})
			assert.Check(t, cmd.Flags().Set("leave-running", leaveRunning))
			assert.Check(t, cmd.Flags().Set("checkpoint-dir", checkpointDir))
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(actualContainerName, containerName))
			expected := checkpoint.CreateOptions{
				CheckpointID:  checkpointName,
				CheckpointDir: checkpointDir,
				Exit:          !tc,
			}
			assert.Check(t, is.Equal(actualOptions, expected))
			assert.Check(t, is.Equal(strings.TrimSpace(cli.OutBuffer().String()), checkpointName))
		})
	}
}
