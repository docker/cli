package swarm

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
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
			swarmLeaveFunc: func() error {
				return errors.Errorf("error leaving the swarm")
			},
			expectedError: "error leaving the swarm",
		},
	}
	for _, tc := range testCases {
		cmd := newLeaveCommand(
			test.NewFakeCli(&fakeClient{
				swarmLeaveFunc: tc.swarmLeaveFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSwarmLeave(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newLeaveCommand(cli)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("Node left the swarm.", strings.TrimSpace(cli.OutBuffer().String())))
}
