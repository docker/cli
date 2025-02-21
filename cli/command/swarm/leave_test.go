package swarm // import "docker.com/cli/v28/cli/command/swarm"

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/v28/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestSwarmLeaveErrors(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		swarmLeaveFunc func() error
		expectedError  string
	}{
		{
			name:          "too-many-args",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
		},
		{
			name: "leave-failed",
			args: []string{},
			swarmLeaveFunc: func() error {
				return errors.New("error leaving the swarm")
			},
			expectedError: "error leaving the swarm",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newLeaveCommand(
				test.NewFakeCli(&fakeClient{
					swarmLeaveFunc: tc.swarmLeaveFunc,
				}))
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestSwarmLeave(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newLeaveCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("Node left the swarm.", strings.TrimSpace(cli.OutBuffer().String())))
}
