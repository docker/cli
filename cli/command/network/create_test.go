package network

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/network"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNetworkCreateErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		networkCreateFunc func(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
		expectedError     string
	}{
		{
			expectedError: "1 argument",
		},
		{
			args: []string{"toto"},
			networkCreateFunc: func(ctx context.Context, name string, createBody network.CreateOptions) (network.CreateResponse, error) {
				return network.CreateResponse{}, errors.New("error creating network")
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
		{
			args: []string{"toto"},
			flags: map[string]string{
				"ip-range": "192.168.83.1-192.168.83.254",
				"gateway":  "192.168.80.1",
				"subnet":   "192.168.80.0/20",
			},
			expectedError: "invalid CIDR address: 192.168.83.1-192.168.83.254",
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
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
		networkCreateFunc: func(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
			assert.Check(t, is.Equal(expectedDriver, options.Driver), "not expected driver error")
			assert.Check(t, is.DeepEqual(expectedOpts, options.IPAM.Config), "not expected driver error")
			return network.CreateResponse{
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

// TestNetworkCreateIPv4 verifies behavior of the "--ipv4" option. This option
// is an optional bool, and must default to "nil", not "true" or "false".
func TestNetworkCreateIPv4(t *testing.T) {
	boolPtr := func(val bool) *bool { return &val }

	tests := []struct {
		doc, name string
		flags     []string
		expected  *bool
	}{
		{
			doc:      "IPv4 default",
			name:     "ipv4-default",
			expected: nil,
		},
		{
			doc:      "IPv4 enabled",
			name:     "ipv4-enabled",
			flags:    []string{"--ipv4=true"},
			expected: boolPtr(true),
		},
		{
			doc:      "IPv4 enabled (shorthand)",
			name:     "ipv4-enabled-shorthand",
			flags:    []string{"--ipv4"},
			expected: boolPtr(true),
		},
		{
			doc:      "IPv4 disabled",
			name:     "ipv4-disabled",
			flags:    []string{"--ipv4=false"},
			expected: boolPtr(false),
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				networkCreateFunc: func(ctx context.Context, name string, createBody network.CreateOptions) (network.CreateResponse, error) {
					assert.Check(t, is.DeepEqual(createBody.EnableIPv4, tc.expected))
					return network.CreateResponse{ID: name}, nil
				},
			})
			cmd := newCreateCommand(cli)
			cmd.SetArgs([]string{tc.name})
			if tc.expected != nil {
				assert.Check(t, cmd.ParseFlags(tc.flags))
			}
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(tc.name, strings.TrimSpace(cli.OutBuffer().String())))
		})
	}
}

// TestNetworkCreateIPv6 verifies behavior of the "--ipv6" option. This option
// is an optional bool, and must default to "nil", not "true" or "false".
func TestNetworkCreateIPv6(t *testing.T) {
	strPtr := func(val bool) *bool { return &val }

	tests := []struct {
		doc, name string
		flags     []string
		expected  *bool
	}{
		{
			doc:      "IPV6 default",
			name:     "ipv6-default",
			expected: nil,
		},
		{
			doc:      "IPV6 enabled",
			name:     "ipv6-enabled",
			flags:    []string{"--ipv6=true"},
			expected: strPtr(true),
		},
		{
			doc:      "IPV6 enabled (shorthand)",
			name:     "ipv6-enabled-shorthand",
			flags:    []string{"--ipv6"},
			expected: strPtr(true),
		},
		{
			doc:      "IPV6 disabled",
			name:     "ipv6-disabled",
			flags:    []string{"--ipv6=false"},
			expected: strPtr(false),
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				networkCreateFunc: func(ctx context.Context, name string, createBody network.CreateOptions) (network.CreateResponse, error) {
					assert.Check(t, is.DeepEqual(tc.expected, createBody.EnableIPv6))
					return network.CreateResponse{ID: name}, nil
				},
			})
			cmd := newCreateCommand(cli)
			cmd.SetArgs([]string{tc.name})
			if tc.expected != nil {
				assert.Check(t, cmd.ParseFlags(tc.flags))
			}
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(tc.name, strings.TrimSpace(cli.OutBuffer().String())))
		})
	}
}
