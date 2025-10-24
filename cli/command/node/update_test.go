package node

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

func TestNodeUpdateErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		nodeInspectFunc func() (client.NodeInspectResult, error)
		nodeUpdateFunc  func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error)
		expectedError   string
	}{
		{
			expectedError: "requires 1 argument",
		},
		{
			args:          []string{"node1", "node2"},
			expectedError: "requires 1 argument",
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
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.NodeLabels(map[string]string{
						"key": "value",
					})),
				}, nil
			},
			flags: map[string]string{
				"label-rm": "not-present",
			},
			expectedError: "key not-present doesn't exist in node's labels",
		},
	}
	for _, tc := range testCases {
		cmd := newUpdateCommand(
			test.NewFakeCli(&fakeClient{
				nodeInspectFunc: tc.nodeInspectFunc,
				nodeUpdateFunc:  tc.nodeUpdateFunc,
			}))
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeUpdate(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		nodeInspectFunc func() (client.NodeInspectResult, error)
		nodeUpdateFunc  func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error)
	}{
		{
			args: []string{"nodeID"},
			flags: map[string]string{
				"role": "manager",
			},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if options.Spec.Role != swarm.NodeRoleManager {
					return client.NodeUpdateResult{}, errors.New("expected role manager, got " + string(options.Spec.Role))
				}
				return client.NodeUpdateResult{}, nil
			},
		},
		{
			args: []string{"nodeID"},
			flags: map[string]string{
				"availability": "drain",
			},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if options.Spec.Availability != swarm.NodeAvailabilityDrain {
					return client.NodeUpdateResult{}, errors.New("expected drain availability, got " + string(options.Spec.Availability))
				}
				return client.NodeUpdateResult{}, nil
			},
		},
		{
			args: []string{"nodeID"},
			flags: map[string]string{
				"label-add": "lbl",
			},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if _, present := options.Spec.Annotations.Labels["lbl"]; !present {
					return client.NodeUpdateResult{}, fmt.Errorf("expected 'lbl' label, got %v", options.Spec.Annotations.Labels)
				}
				return client.NodeUpdateResult{}, nil
			},
		},
		{
			args: []string{"nodeID"},
			flags: map[string]string{
				"label-add": "key=value",
			},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if value, present := options.Spec.Annotations.Labels["key"]; !present || value != "value" {
					return client.NodeUpdateResult{}, fmt.Errorf("expected 'key' label to be 'value', got %v", options.Spec.Annotations.Labels)
				}
				return client.NodeUpdateResult{}, nil
			},
		},
		{
			args: []string{"nodeID"},
			flags: map[string]string{
				"label-rm": "key",
			},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.NodeLabels(map[string]string{
						"key": "value",
					})),
				}, nil
			},
			nodeUpdateFunc: func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
				if len(options.Spec.Annotations.Labels) > 0 {
					return client.NodeUpdateResult{}, fmt.Errorf("expected no labels, got %v", options.Spec.Annotations.Labels)
				}
				return client.NodeUpdateResult{}, nil
			},
		},
	}
	for _, tc := range testCases {
		cmd := newUpdateCommand(
			test.NewFakeCli(&fakeClient{
				nodeInspectFunc: tc.nodeInspectFunc,
				nodeUpdateFunc:  tc.nodeUpdateFunc,
			}))
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		assert.NilError(t, cmd.Execute())
	}
}
