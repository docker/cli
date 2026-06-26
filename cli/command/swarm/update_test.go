package swarm

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestSwarmUpdateErrors(t *testing.T) {
	testCases := []struct {
		name                  string
		args                  []string
		flags                 map[string]string
		swarmInspectFunc      func() (client.SwarmInspectResult, error)
		swarmUpdateFunc       func(client.SwarmUpdateOptions) (client.SwarmUpdateResult, error)
		swarmGetUnlockKeyFunc func() (client.SwarmGetUnlockKeyResult, error)
		expectedError         string
	}{
		{
			name:          "too-many-args",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
		},
		{
			name: "swarm-inspect-error",
			flags: map[string]string{
				flagTaskHistoryLimit: "10",
			},
			swarmInspectFunc: func() (client.SwarmInspectResult, error) {
				return client.SwarmInspectResult{}, errors.New("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "swarm-update-error",
			flags: map[string]string{
				flagTaskHistoryLimit: "10",
			},
			swarmUpdateFunc: func(client.SwarmUpdateOptions) (client.SwarmUpdateResult, error) {
				return client.SwarmUpdateResult{}, errors.New("error updating the swarm")
			},
			expectedError: "error updating the swarm",
		},
		{
			name: "swarm-unlock-key-error",
			flags: map[string]string{
				flagAutolock: "true",
			},
			swarmInspectFunc: func() (client.SwarmInspectResult, error) {
				return client.SwarmInspectResult{
					Swarm: *builders.Swarm(),
				}, nil
			},
			swarmGetUnlockKeyFunc: func() (client.SwarmGetUnlockKeyResult, error) {
				return client.SwarmGetUnlockKeyResult{}, errors.New("error getting unlock key")
			},
			expectedError: "error getting unlock key",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newUpdateCommand(
				test.NewFakeCli(&fakeClient{
					swarmInspectFunc:      tc.swarmInspectFunc,
					swarmUpdateFunc:       tc.swarmUpdateFunc,
					swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
				}))
			if tc.args == nil {
				cmd.SetArgs([]string{})
			} else {
				cmd.SetArgs(tc.args)
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			for k, v := range tc.flags {
				assert.Check(t, cmd.Flags().Set(k, v))
			}
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestSwarmUpdate(t *testing.T) {
	swarmInfo := builders.Swarm()
	swarmInfo.ClusterInfo.TLSInfo.TrustRoot = "trust-root"

	testCases := []struct {
		name                  string
		args                  []string
		flags                 map[string]string
		swarmInspectFunc      func() (client.SwarmInspectResult, error)
		swarmUpdateFunc       func(client.SwarmUpdateOptions) (client.SwarmUpdateResult, error)
		swarmGetUnlockKeyFunc func() (client.SwarmGetUnlockKeyResult, error)
	}{
		{
			name: "noargs",
		},
		{
			name: "all-flags-quiet",
			flags: map[string]string{
				flagTaskHistoryLimit:    "10",
				flagDispatcherHeartbeat: "10s",
				flagCertExpiry:          "20s",
				flagExternalCA:          "protocol=cfssl,url=https://example.com.",
				flagMaxSnapshots:        "10",
				flagSnapshotInterval:    "100",
				flagAutolock:            "true",
			},
			swarmInspectFunc: func() (client.SwarmInspectResult, error) {
				return client.SwarmInspectResult{
					Swarm: *swarmInfo,
				}, nil
			},
			swarmUpdateFunc: func(options client.SwarmUpdateOptions) (client.SwarmUpdateResult, error) {
				if *options.Spec.Orchestration.TaskHistoryRetentionLimit != 10 {
					return client.SwarmUpdateResult{}, errors.New("historyLimit not correctly set")
				}
				heartbeatDuration, err := time.ParseDuration("10s")
				if err != nil {
					return client.SwarmUpdateResult{}, err
				}
				if options.Spec.Dispatcher.HeartbeatPeriod != heartbeatDuration {
					return client.SwarmUpdateResult{}, errors.New("heartbeatPeriodLimit not correctly set")
				}
				certExpiryDuration, err := time.ParseDuration("20s")
				if err != nil {
					return client.SwarmUpdateResult{}, err
				}
				if options.Spec.CAConfig.NodeCertExpiry != certExpiryDuration {
					return client.SwarmUpdateResult{}, errors.New("certExpiry not correctly set")
				}
				if len(options.Spec.CAConfig.ExternalCAs) != 1 || options.Spec.CAConfig.ExternalCAs[0].CACert != "trust-root" {
					return client.SwarmUpdateResult{}, errors.New("externalCA not correctly set")
				}
				if *options.Spec.Raft.KeepOldSnapshots != 10 {
					return client.SwarmUpdateResult{}, errors.New("keepOldSnapshots not correctly set")
				}
				if options.Spec.Raft.SnapshotInterval != 100 {
					return client.SwarmUpdateResult{}, errors.New("snapshotInterval not correctly set")
				}
				if !options.Spec.EncryptionConfig.AutoLockManagers {
					return client.SwarmUpdateResult{}, errors.New("auto-lock not correctly set")
				}
				return client.SwarmUpdateResult{}, nil
			},
		},
		{
			name: "auto-lock-unlock-key",
			flags: map[string]string{
				flagTaskHistoryLimit: "10",
				flagAutolock:         "true",
			},
			swarmUpdateFunc: func(options client.SwarmUpdateOptions) (client.SwarmUpdateResult, error) {
				if *options.Spec.Orchestration.TaskHistoryRetentionLimit != 10 {
					return client.SwarmUpdateResult{}, errors.New("historyLimit not correctly set")
				}
				return client.SwarmUpdateResult{}, nil
			},
			swarmInspectFunc: func() (client.SwarmInspectResult, error) {
				return client.SwarmInspectResult{
					Swarm: *builders.Swarm(),
				}, nil
			},
			swarmGetUnlockKeyFunc: func() (client.SwarmGetUnlockKeyResult, error) {
				return client.SwarmGetUnlockKeyResult{
					Key: "unlock-key",
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				swarmInspectFunc:      tc.swarmInspectFunc,
				swarmUpdateFunc:       tc.swarmUpdateFunc,
				swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
			})
			cmd := newUpdateCommand(cli)
			if tc.args == nil {
				cmd.SetArgs([]string{})
			} else {
				cmd.SetArgs(tc.args)
			}
			for k, v := range tc.flags {
				assert.Check(t, cmd.Flags().Set(k, v))
			}
			cmd.SetOut(cli.OutBuffer())
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("update-%s.golden", tc.name))
		})
	}
}
