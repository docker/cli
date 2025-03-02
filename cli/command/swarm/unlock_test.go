package swarm

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"gotest.tools/v3/assert"
)

func TestSwarmUnlockErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		swarmUnlockFunc func(req swarm.UnlockRequest) error
		infoFunc        func() (system.Info, error)
		expectedError   string
	}{
		{
			name:          "too-many-args",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
		},
		{
			name: "is-not-part-of-a-swarm",
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						LocalNodeState: swarm.LocalNodeStateInactive,
					},
				}, nil
			},
			expectedError: "This node is not part of a swarm",
		},
		{
			name: "is-not-locked",
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						LocalNodeState: swarm.LocalNodeStateActive,
					},
				}, nil
			},
			expectedError: "Error: swarm is not locked",
		},
		{
			name: "unlockrequest-failed",
			infoFunc: func() (system.Info, error) {
				return system.Info{
					Swarm: swarm.Info{
						LocalNodeState: swarm.LocalNodeStateLocked,
					},
				}, nil
			},
			swarmUnlockFunc: func(req swarm.UnlockRequest) error {
				return errors.New("error unlocking the swarm")
			},
			expectedError: "error unlocking the swarm",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newUnlockCommand(
				test.NewFakeCli(&fakeClient{
					infoFunc:        tc.infoFunc,
					swarmUnlockFunc: tc.swarmUnlockFunc,
				}))
			if tc.args == nil {
				cmd.SetArgs([]string{})
			} else {
				cmd.SetArgs(tc.args)
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestSwarmUnlock(t *testing.T) {
	input := "unlockKey"
	dockerCli := test.NewFakeCli(&fakeClient{
		infoFunc: func() (system.Info, error) {
			return system.Info{
				Swarm: swarm.Info{
					LocalNodeState: swarm.LocalNodeStateLocked,
				},
			}, nil
		},
		swarmUnlockFunc: func(req swarm.UnlockRequest) error {
			if req.UnlockKey != input {
				return errors.New("invalid unlock key")
			}
			return nil
		},
	})
	dockerCli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader(input))))
	cmd := newUnlockCommand(dockerCli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())
}
