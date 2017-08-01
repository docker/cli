package swarm

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/pkg/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
			expectedError: "accepts no argument(s)",
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
		buf := new(bytes.Buffer)
		cmd := newLeaveCommand(
			test.NewFakeCliWithOutput(&fakeClient{
				swarmLeaveFunc: tc.swarmLeaveFunc,
			}, buf))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSwarmLeave(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := newLeaveCommand(
		test.NewFakeCliWithOutput(&fakeClient{}, buf))
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "Node left the swarm.", strings.TrimSpace(buf.String()))
}
