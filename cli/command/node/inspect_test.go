package node

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNodeInspectErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		nodeInspectFunc func() (client.NodeInspectResult, error)
		infoFunc        func() (client.SystemInfoResult, error)
		expectedError   string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"self"},
			infoFunc: func() (client.SystemInfoResult, error) {
				return client.SystemInfoResult{}, errors.New("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting the node")
			},
			infoFunc: func() (client.SystemInfoResult, error) {
				return client.SystemInfoResult{}, errors.New("error asking for node info")
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"self"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting the node")
			},
			infoFunc: func() (client.SystemInfoResult, error) {
				return client.SystemInfoResult{
					Info: system.Info{
						Swarm: swarm.Info{NodeID: "abc"},
					},
				}, nil
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"self"},
			flags: map[string]string{
				"pretty": "true",
			},
			infoFunc: func() (client.SystemInfoResult, error) {
				return client.SystemInfoResult{}, errors.New("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
	}
	for _, tc := range testCases {
		cmd := newInspectCommand(
			test.NewFakeCli(&fakeClient{
				nodeInspectFunc: tc.nodeInspectFunc,
				infoFunc:        tc.infoFunc,
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

func TestNodeInspectPretty(t *testing.T) {
	testCases := []struct {
		name            string
		nodeInspectFunc func() (client.NodeInspectResult, error)
	}{
		{
			name: "simple",
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.NodeLabels(map[string]string{"lbl1": "value1"})),
				}, nil
			},
		},
		{
			name: "manager",
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.Manager()),
				}, nil
			},
		},
		{
			name: "manager-leader",
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(builders.Manager(builders.Leader())),
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				nodeInspectFunc: tc.nodeInspectFunc,
			})
			cmd := newInspectCommand(cli)
			cmd.SetArgs([]string{"nodeID"})
			assert.Check(t, cmd.Flags().Set("pretty", "true"))
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("node-inspect-pretty.%s.golden", tc.name))
		})
	}
}
