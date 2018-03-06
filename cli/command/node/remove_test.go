package node

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/pkg/errors"
)

func TestNodeRemoveErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		nodeRemoveFunc func() error
		expectedError  string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"nodeID"},
			nodeRemoveFunc: func() error {
				return errors.Errorf("error removing the node")
			},
			expectedError: "error removing the node",
		},
	}
	for _, tc := range testCases {
		cmd := newRemoveCommand(
			test.NewFakeCli(&fakeClient{
				nodeRemoveFunc: tc.nodeRemoveFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeRemoveMultiple(t *testing.T) {
	cmd := newRemoveCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"nodeID1", "nodeID2"})
	assert.NilError(t, cmd.Execute())
}
