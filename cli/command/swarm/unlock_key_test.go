package swarm

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestSwarmUnlockKeyErrors(t *testing.T) {
	testCases := []struct {
		name                  string
		args                  []string
		flags                 map[string]string
		swarmInspectFunc      func() (swarm.Swarm, error)
		swarmUpdateFunc       func(swarm swarm.Spec, flags swarm.UpdateFlags) error
		swarmGetUnlockKeyFunc func() (types.SwarmUnlockKeyResponse, error)
		expectedError         string
	}{
		{
			name:          "too-many-args",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
		},
		{
			name: "swarm-inspect-rotate-failed",
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return swarm.Swarm{}, errors.Errorf("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "swarm-rotate-no-autolock-failed",
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *Swarm(), nil
			},
			expectedError: "cannot rotate because autolock is not turned on",
		},
		{
			name: "swarm-update-failed",
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *Swarm(Autolock()), nil
			},
			swarmUpdateFunc: func(swarm swarm.Spec, flags swarm.UpdateFlags) error {
				return errors.Errorf("error updating the swarm")
			},
			expectedError: "error updating the swarm",
		},
		{
			name: "swarm-get-unlock-key-failed",
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{}, errors.Errorf("error getting unlock key")
			},
			expectedError: "error getting unlock key",
		},
		{
			name: "swarm-no-unlock-key-failed",
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{
					UnlockKey: "",
				}, nil
			},
			expectedError: "no unlock key is set",
		},
	}
	for _, tc := range testCases {
		cmd := newUnlockKeyCommand(
			test.NewFakeCli(&fakeClient{
				swarmInspectFunc:      tc.swarmInspectFunc,
				swarmUpdateFunc:       tc.swarmUpdateFunc,
				swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
			}))
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestSwarmUnlockKey(t *testing.T) {
	testCases := []struct {
		name                  string
		args                  []string
		flags                 map[string]string
		swarmInspectFunc      func() (swarm.Swarm, error)
		swarmUpdateFunc       func(swarm swarm.Spec, flags swarm.UpdateFlags) error
		swarmGetUnlockKeyFunc func() (types.SwarmUnlockKeyResponse, error)
	}{
		{
			name: "unlock-key",
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{
					UnlockKey: "unlock-key",
				}, nil
			},
		},
		{
			name: "unlock-key-quiet",
			flags: map[string]string{
				flagQuiet: "true",
			},
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{
					UnlockKey: "unlock-key",
				}, nil
			},
		},
		{
			name: "unlock-key-rotate",
			flags: map[string]string{
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *Swarm(Autolock()), nil
			},
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{
					UnlockKey: "unlock-key",
				}, nil
			},
		},
		{
			name: "unlock-key-rotate-quiet",
			flags: map[string]string{
				flagQuiet:  "true",
				flagRotate: "true",
			},
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return *Swarm(Autolock()), nil
			},
			swarmGetUnlockKeyFunc: func() (types.SwarmUnlockKeyResponse, error) {
				return types.SwarmUnlockKeyResponse{
					UnlockKey: "unlock-key",
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			swarmInspectFunc:      tc.swarmInspectFunc,
			swarmUpdateFunc:       tc.swarmUpdateFunc,
			swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
		})
		cmd := newUnlockKeyCommand(cli)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("unlockkeys-%s.golden", tc.name))
	}
}
