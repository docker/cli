package network

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestNetworkCreateErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		networkCreateFunc func(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
		expectedError     string
	}{
		{
			expectedError: "exactly 1 argument",
		},
		{
			args: []string{"toto"},
			networkCreateFunc: func(ctx context.Context, name string, createBody types.NetworkCreate) (types.NetworkCreateResponse, error) {
				return types.NetworkCreateResponse{}, errors.Errorf("error creating network")
			},
			expectedError: "error creating network",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.255.0.0/24",
				"gateway":  "255.0.255.0/24",
				"subnet":   "10.1.2.0.30.50",
			},
			expectedError: "invalid CIDR address: 10.1.2.0.30.50",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.255.0.0.30/24",
				"gateway":  "255.0.255.0/24",
				"subnet":   "255.0.0.0/24",
			},
			expectedError: "invalid CIDR address: 255.255.0.0.30/24",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"gateway": "255.0.0.0/24",
			},
			expectedError: "every ip-range or gateway must have a corresponding subnet",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.0.0.0/24",
			},
			expectedError: "every ip-range or gateway must have a corresponding subnet",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.0.0.0/24",
				"gateway":  "255.0.0.0/24",
			},
			expectedError: "every ip-range or gateway must have a corresponding subnet",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.255.0.0/24",
				"gateway":  "255.0.255.0/24",
				"subnet":   "10.1.2.0/23,10.1.3.248/30",
			},
			expectedError: "multiple overlapping subnet configuration is not supported",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "192.168.1.0/24,192.168.1.200/24",
				"gateway":  "192.168.1.1,192.168.1.4",
				"subnet":   "192.168.2.0/24,192.168.1.250/24",
			},
			expectedError: "cannot configure multiple ranges (192.168.1.200/24, 192.168.1.0/24) on the same subnet (192.168.1.250/24)",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "255.255.200.0/24,255.255.120.0/24",
				"gateway":  "255.0.255.0/24",
				"subnet":   "255.255.255.0/24,255.255.0.255/24",
			},
			expectedError: "no matching subnet for range 255.255.200.0/24",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "192.168.1.0/24",
				"gateway":  "192.168.1.1,192.168.1.4",
				"subnet":   "192.168.2.0/24,192.168.1.250/24",
			},
			expectedError: "cannot configure multiple gateways (192.168.1.4, 192.168.1.1) for the same subnet (192.168.1.250/24)",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "192.168.1.0/24",
				"gateway":  "192.168.4.1,192.168.5.4",
				"subnet":   "192.168.2.0/24,192.168.1.250/24",
			},
			expectedError: "no matching subnet for gateway 192.168.4.1",
		},
		{
			args: []string{"toto"},
			flags: map[string]string{
				"gateway":     "255.255.0.0/24",
				"subnet":      "255.255.0.0/24",
				"aux-address": "255.255.0.30/24",
			},
			expectedError: "no matching subnet for aux-address",
		},
	}

	for _, tc := range testCases {
		cmd := newCreateCommand(
			test.NewFakeCli(&fakeClient{
				networkCreateFunc: tc.networkCreateFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.NilError(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)

	}
}
func TestNetworkCreateWithFlags(t *testing.T) {
	expectedDriver := "foo"
	expectedOpts := []network.IPAMConfig{
		{
			Subnet:     "192.168.4.0/24",
			IPRange:    "192.168.4.0/24",
			Gateway:    "192.168.4.1/24",
			AuxAddress: map[string]string{},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		networkCreateFunc: func(ctx context.Context, name string, createBody types.NetworkCreate) (types.NetworkCreateResponse, error) {
			assert.Check(t, is.Equal(expectedDriver, createBody.Driver), "not expected driver error")
			assert.Check(t, is.DeepEqual(expectedOpts, createBody.IPAM.Config), "not expected driver error")
			return types.NetworkCreateResponse{
				ID: name,
			}, nil
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
	assert.Check(t, is.Equal("banana", strings.TrimSpace(cli.OutBuffer().String())))
}
