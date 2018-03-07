package node

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
)

func TestNodePromoteErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		nodeInspectFunc func() (swarm.Node, []byte, error)
		nodeUpdateFunc  func(nodeID string, version swarm.Version, node swarm.NodeSpec) error
		expectedError   string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return swarm.Node{}, []byte{}, errors.Errorf("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"nodeID"},
			nodeUpdateFunc: func(nodeID string, version swarm.Version, node swarm.NodeSpec) error {
				return errors.Errorf("error updating the node")
			},
			expectedError: "error updating the node",
		},
	}
	for _, tc := range testCases {
		cmd := newPromoteCommand(
			test.NewFakeCli(&fakeClient{
				nodeInspectFunc: tc.nodeInspectFunc,
				nodeUpdateFunc:  tc.nodeUpdateFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodePromoteNoChange(t *testing.T) {
	cmd := newPromoteCommand(
		test.NewFakeCli(&fakeClient{
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *Node(Manager()), []byte{}, nil
			},
			nodeUpdateFunc: func(nodeID string, version swarm.Version, node swarm.NodeSpec) error {
				if node.Role != swarm.NodeRoleManager {
					return errors.Errorf("expected role manager, got %s", node.Role)
				}
				return nil
			},
		}))
	cmd.SetArgs([]string{"nodeID"})
	assert.NilError(t, cmd.Execute())
}

func TestNodePromoteMultipleNode(t *testing.T) {
	cmd := newPromoteCommand(
		test.NewFakeCli(&fakeClient{
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *Node(), []byte{}, nil
			},
			nodeUpdateFunc: func(nodeID string, version swarm.Version, node swarm.NodeSpec) error {
				if node.Role != swarm.NodeRoleManager {
					return errors.Errorf("expected role manager, got %s", node.Role)
				}
				return nil
			},
		}))
	cmd.SetArgs([]string{"nodeID1", "nodeID2"})
	assert.NilError(t, cmd.Execute())
}
