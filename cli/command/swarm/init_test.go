package swarm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestSwarmInitErrorOnAPIFailure(t *testing.T) {
	testCases := []struct {
		name                  string
		flags                 map[string]string
		swarmInitFunc         func(swarm.InitRequest) (string, error)
		swarmInspectFunc      func() (swarm.Swarm, error)
		swarmGetUnlockKeyFunc func() (swarm.UnlockKeyResponse, error)
		nodeInspectFunc       func() (swarm.Node, []byte, error)
		expectedError         string
	}{
		{
			name: "init-failed",
			swarmInitFunc: func(swarm.InitRequest) (string, error) {
				return "", errors.New("error initializing the swarm")
			},
			expectedError: "error initializing the swarm",
		},
		{
			name: "init-failed-with-ip-choice",
			swarmInitFunc: func(swarm.InitRequest) (string, error) {
				return "", errors.New("could not choose an IP address to advertise")
			},
			expectedError: "could not choose an IP address to advertise - specify one with --advertise-addr",
		},
		{
			name: "swarm-inspect-after-init-failed",
			swarmInspectFunc: func() (swarm.Swarm, error) {
				return swarm.Swarm{}, errors.New("error inspecting the swarm")
			},
			expectedError: "error inspecting the swarm",
		},
		{
			name: "node-inspect-after-init-failed",
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return swarm.Node{}, []byte{}, errors.New("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			name: "swarm-get-unlock-key-after-init-failed",
			flags: map[string]string{
				flagAutolock: "true",
			},
			swarmGetUnlockKeyFunc: func() (swarm.UnlockKeyResponse, error) {
				return swarm.UnlockKeyResponse{}, errors.New("error getting swarm unlock key")
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
		swarmInitFunc         func(req swarm.InitRequest) (string, error)
		swarmInspectFunc      func() (swarm.Swarm, error)
		swarmGetUnlockKeyFunc func() (swarm.UnlockKeyResponse, error)
		nodeInspectFunc       func() (swarm.Node, []byte, error)
	}{
		{
			name: "init",
			swarmInitFunc: func(swarm.InitRequest) (string, error) {
				return "nodeID", nil
			},
		},
		{
			name: "init-autolock",
			flags: map[string]string{
				flagAutolock: "true",
			},
			swarmInitFunc: func(swarm.InitRequest) (string, error) {
				return "nodeID", nil
			},
			swarmGetUnlockKeyFunc: func() (swarm.UnlockKeyResponse, error) {
				return swarm.UnlockKeyResponse{
					UnlockKey: "unlock-key",
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				swarmInitFunc:         tc.swarmInitFunc,
				swarmInspectFunc:      tc.swarmInspectFunc,
				swarmGetUnlockKeyFunc: tc.swarmGetUnlockKeyFunc,
				nodeInspectFunc:       tc.nodeInspectFunc,
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
		swarmInitFunc: func(req swarm.InitRequest) (string, error) {
			if assert.Check(t, is.Len(req.Spec.CAConfig.ExternalCAs, 1)) {
				assert.Equal(t, req.Spec.CAConfig.ExternalCAs[0].CACert, cert)
				assert.Equal(t, req.Spec.CAConfig.ExternalCAs[0].Protocol, swarm.ExternalCAProtocolCFSSL)
				assert.Equal(t, req.Spec.CAConfig.ExternalCAs[0].URL, "https://example.com")
			}
			return "nodeID", nil
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
