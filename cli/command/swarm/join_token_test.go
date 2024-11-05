package swarm

import (
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestSwarmJoinTokenErrors(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		flags            map[string]string
		infoFunc         func() (system.Info, error)
		swarmInspectFunc func() (swarm.Swarm, error)
		swarmUpdateFunc  func(swarm swarm.Spec, flags swarm.UpdateFlags) error
		nodeInspectFunc  func() (swarm.Node, []byte, error)
		expectedError    string
	}{
		{
			name:          "not-enough-args",
			expectedError: "requires 1 argument",
		},
		{
			name:          "too-many-args",
			args:          []string{"worker", "manager"},
			expectedError: "requires 1 argument",
		},
		{
			name:          "invalid-args",
			args:          []string{"foo"},
			expectedError: "unknown role foo",
		},
		{
			name: "swarm-inspect-failed",
			args: []string{"worker"},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return swarm.Swarm{}, errors.Errorf("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "swarm-inspect-rotate-failed",
			args: []string{"worker"},
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return swarm.Swarm{}, errors.Errorf("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "swarm-update-failed",
			args: []string{"worker"},
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmUpdateFunc: func(swarm swarm.Spec, flags swarm.UpdateFlags) error {
				return errors.Errorf("error updating the swarm")
			},
			expectedError: "error updating the swarm",
		},
		{
			name: "node-inspect-failed",
			args: []string{"worker"},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return swarm.Node{}, []byte{}, errors.Errorf("error inspecting node")
			},
			expectedError: "error inspecting node",
		},
		{
			name: "info-failed",
			args: []string{"worker"},
			infoFunc: func() (system.Info, error) {
				return system.Info{}, errors.Errorf("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				swarmInspectFunc: tc.swarmInspectFunc,
				swarmUpdateFunc:  tc.swarmUpdateFunc,
				infoFunc:         tc.infoFunc,
				nodeInspectFunc:  tc.nodeInspectFunc,
			})
			cmd := newJoinTokenCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestSwarmJoinToken(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		flags            map[string]string
		infoFunc         func() (system.Info, error)
		swarmInspectFunc func() (swarm.Swarm, error)
		nodeInspectFunc  func() (swarm.Node, []byte, error)
	}{
		{
			name: "worker",
			args: []string{"worker"},
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						NodeID: "nodeID",
					},
				}, nil
			},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(builders.Manager()), []byte{}, nil
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *builders.Swarm(), nil
			},
		},
		{
			name: "manager",
			args: []string{"manager"},
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						NodeID: "nodeID",
					},
				}, nil
			},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(builders.Manager()), []byte{}, nil
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *builders.Swarm(), nil
			},
		},
		{
			name: "manager-rotate",
			args: []string{"manager"},
			flags: map[string]string{
				flagRotate: "true",
			},
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						NodeID: "nodeID",
					},
				}, nil
			},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(builders.Manager()), []byte{}, nil
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *builders.Swarm(), nil
			},
		},
		{
			name: "worker-quiet",
			args: []string{"worker"},
			flags: map[string]string{
				flagQuiet: "true",
			},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(builders.Manager()), []byte{}, nil
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *builders.Swarm(), nil
			},
		},
		{
			name: "manager-quiet",
			args: []string{"manager"},
			flags: map[string]string{
				flagQuiet: "true",
			},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(builders.Manager()), []byte{}, nil
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *builders.Swarm(), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				swarmInspectFunc: tc.swarmInspectFunc,
				infoFunc:         tc.infoFunc,
				nodeInspectFunc:  tc.nodeInspectFunc,
			})
			cmd := newJoinTokenCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("jointoken-%s.golden", tc.name))
		})
	}
}
