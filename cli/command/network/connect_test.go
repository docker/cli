package network

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/network"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNetworkConnectErrors(t *testing.T) {
	testCases := []struct {
		args               []string
		networkConnectFunc func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
		expectedError      string
	}{
		{
			expectedError: "requires 2 arguments",
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNetworkConnectWithFlags(t *testing.T) {
	expectedConfig := &network.EndpointSettings{
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address:  "192.168.4.1",
			IPv6Address:  "fdef:f401:8da0:1234::5678",
			LinkLocalIPs: []string{"169.254.42.42"},
		},
		Links:   []string{"otherctr"},
		Aliases: []string{"poor-yorick"},
		DriverOpts: map[string]string{
			"driveropt1": "optval1,optval2",
			"driveropt2": "optval4",
		},
		GwPriority: 100,
	}
	cli := test.NewFakeCli(&fakeClient{
		networkConnectFunc: func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
			assert.Check(t, is.DeepEqual(expectedConfig, config))
			return nil
		},
	})
	args := []string{"mynet", "myctr"}
	cmd := newConnectCommand(cli)

	cmd.SetArgs(args)
	for _, opt := range []struct{ name, value string }{
		{"alias", "poor-yorick"},
		{"driver-opt", "\"driveropt1=optval1,optval2\""},
		{"driver-opt", "driveropt2=optval3"},
		{"driver-opt", "driveropt2=optval4"}, // replaces value
		{"ip", "192.168.4.1"},
		{"ip6", "fdef:f401:8da0:1234::5678"},
		{"link", "otherctr"},
		{"link-local-ip", "169.254.42.42"},
		{"gw-priority", "100"},
	} {
		err := cmd.Flags().Set(opt.name, opt.value)
		assert.Check(t, err)
	}
	assert.NilError(t, cmd.Execute())
}
