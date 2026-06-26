package swarm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestSwarmInitErrorOnAPIFailure(t *testing.T) {
	testCases := []struct {
		name                  string
		flags                 map[string]string
		swarmInitFunc         func(client.SwarmInitOptions) (client.SwarmInitResult, error)
		swarmInspectFunc      func() (client.SwarmInspectResult, error)
		swarmGetUnlockKeyFunc func() (client.SwarmGetUnlockKeyResult, error)
		nodeInspectFunc       func() (client.NodeInspectResult, error)
		expectedError         string
	}{
		{
			name: "init-failed",
			swarmInitFunc: func(client.SwarmInitOptions) (client.SwarmInitResult, error) {
				return client.SwarmInitResult{}, errors.New("error initializing the swarm")
			},
			expectedError: "error initializing the swarm",
		},
		{
			name: "init-failed-with-ip-choice",
			swarmInitFunc: func(client.SwarmInitOptions) (client.SwarmInitResult, error) {
				return client.SwarmInitResult{}, errors.New("could not choose an IP address to advertise")
			},
			expectedError: "could not choose an IP address to advertise - specify one with --advertise-addr",
		},
		{
			name: "swarm-inspect-after-init-failed",
			swarmInspectFunc: func() (client.SwarmInspectResult, error) {
				return client.SwarmInspectResult{}, errors.New("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "node-inspect-after-init-failed",
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			name: "swarm-get-unlock-key-after-init-failed",
			flags: map[string]string{
				flagAutolock: "true",
			},
			swarmGetUnlockKeyFunc: func() (client.SwarmGetUnlockKeyResult, error) {
				return client.SwarmGetUnlockKeyResult{}, errors.New("error getting swarm unlock key")
			},
			expectedError: "could not fetch unlock key: error getting swarm unlock key",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newInitCommand(
				test.NewFakeCli(&fakeClient{
					swarmInitFunc:         tc.swarmInitFunc,
					swarmInspectFunc:      tc.swarmInspectFunc,
					swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
					nodeInspectFunc:       tc.nodeInspectFunc,
				}))
			cmd.SetArgs([]string{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			for k, v := range tc.flags {
				assert.Check(t, cmd.Flags().Set(k, v))
			}
			assert.Error(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestSwarmInit(t *testing.T) {
	testCases := []struct {
		name                  string
		flags                 map[string]string
		swarmInitFunc         func(client.SwarmInitOptions) (client.SwarmInitResult, error)
		swarmGetUnlockKeyFunc func() (client.SwarmGetUnlockKeyResult, error)
	}{
		{
			name: "init",
			swarmInitFunc: func(client.SwarmInitOptions) (client.SwarmInitResult, error) {
				return client.SwarmInitResult{NodeID: "nodeID"}, nil
			},
		},
		{
			name: "init-auto-lock",
			flags: map[string]string{
				flagAutolock: "true",
			},
			swarmInitFunc: func(client.SwarmInitOptions) (client.SwarmInitResult, error) {
				return client.SwarmInitResult{NodeID: "nodeID"}, nil
			},
			swarmGetUnlockKeyFunc: func() (client.SwarmGetUnlockKeyResult, error) {
				return client.SwarmGetUnlockKeyResult{Key: "unlock-key"}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				swarmInitFunc:         tc.swarmInitFunc,
				swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
			})
			cmd := newInitCommand(cli)
			cmd.SetArgs([]string{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			for k, v := range tc.flags {
				assert.Check(t, cmd.Flags().Set(k, v))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("init-%s.golden", tc.name))
		})
	}
}

func TestSwarmInitWithExternalCA(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		swarmInitFunc: func(options client.SwarmInitOptions) (client.SwarmInitResult, error) {
			if assert.Check(t, is.Len(options.Spec.CAConfig.ExternalCAs, 1)) {
				assert.Equal(t, options.Spec.CAConfig.ExternalCAs[0].CACert, cert)
				assert.Equal(t, options.Spec.CAConfig.ExternalCAs[0].Protocol, swarm.ExternalCAProtocolCFSSL)
				assert.Equal(t, options.Spec.CAConfig.ExternalCAs[0].URL, "https://example.com")
			}
			return client.SwarmInitResult{NodeID: "nodeID"}, nil
		},
	})

	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	err := os.WriteFile(certFile, []byte(cert), 0o644)
	assert.NilError(t, err)

	cmd := newInitCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Flags().Set(flagExternalCA, "protocol=cfssl,url=https://example.com,cacert="+certFile))
	assert.NilError(t, cmd.Execute())
}
