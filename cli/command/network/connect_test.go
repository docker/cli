package network

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/network"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestNetworkConnectErrors(t *testing.T) {
	testCases := []struct {
		args               []string
		networkConnectFunc func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
		expectedError      string
	}{
		{
			expectedError: "requires exactly 2 arguments",
		},
		{
			args: []string{"toto", "titi"},
			networkConnectFunc: func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
				return errors.Errorf("error connecting network")
			},
			expectedError: "error connecting network",
		},
	}

	for _, tc := range testCases {
		cmd := newConnectCommand(
			test.NewFakeCli(&fakeClient{
				networkConnectFunc: tc.networkConnectFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)

	}
}

func TestNetworkConnectWithFlags(t *testing.T) {
	expectedOpts := []network.IPAMConfig{
		{
			Subnet:     "192.168.4.0/24",
			IPRange:    "192.168.4.0/24",
			Gateway:    "192.168.4.1/24",
			AuxAddress: map[string]string{},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		networkConnectFunc: func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
			assert.Check(t, is.DeepEqual(expectedOpts, config.IPAMConfig), "not expected driver error")
			return nil
		},
	})
	args := []string{"banana"}
	cmd := newCreateCommand(cli)

	cmd.SetArgs(args)
	cmd.Flags().Set("driver", "foo")
	cmd.Flags().Set("ip-range", "192.168.4.0/24")
	cmd.Flags().Set("gateway", "192.168.4.1/24")
	cmd.Flags().Set("subnet", "192.168.4.0/24")
	assert.NilError(t, cmd.Execute())
}
