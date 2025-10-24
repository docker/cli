package node

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestNodePromoteErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		nodeInspectFunc func() (client.NodeInspectResult, error)
		nodeUpdateFunc  func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error)
		expectedError   string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"nodeID"},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				return client.NodeUpdateResult{}, errors.New("error updating the node")
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodePromoteNoChange(t *testing.T) {
	cmd := newPromoteCommand(
		test.NewFakeCli(&fakeClient{
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.Manager()),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if options.Spec.Role != swarm.NodeRoleManager {
					return client.NodeUpdateResult{}, errors.New("expected role manager, got" + string(options.Spec.Role))
				}
				return client.NodeUpdateResult{}, nil
			},
		}))
	cmd.SetArgs([]string{"nodeID"})
	assert.NilError(t, cmd.Execute())
}

func TestNodePromoteMultipleNode(t *testing.T) {
	cmd := newPromoteCommand(
		test.NewFakeCli(&fakeClient{
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if options.Spec.Role != swarm.NodeRoleManager {
					return client.NodeUpdateResult{}, errors.New("expected role manager, got" + string(options.Spec.Role))
				}
				return client.NodeUpdateResult{}, nil
			},
		}))
	cmd.SetArgs([]string{"nodeID1", "nodeID2"})
	assert.NilError(t, cmd.Execute())
}
