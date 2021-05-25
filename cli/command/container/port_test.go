package container

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewPortCommandOutput(t *testing.T) {
	testCases := []struct {
		name string
		ips  []string
	}{
		{
			name: "container-port-ipv4",
			ips:  []string{"0.0.0.0"},
		},
		{
			name: "container-port-ipv6",
			ips:  []string{"::"},
		},
		{
			name: "container-port-ipv6-and-ipv4",
			ips:  []string{"::", "0.0.0.0"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				inspectFunc: func(string) (types.ContainerJSON, error) {
					ci := types.ContainerJSON{NetworkSettings: &types.NetworkSettings{}}
					ci.NetworkSettings.Ports = nat.PortMap{
						"80/tcp": make([]nat.PortBinding, len(tc.ips)),
					}
					for i, ip := range tc.ips {
						ci.NetworkSettings.Ports["80/tcp"][i] = nat.PortBinding{
							HostIP: ip, HostPort: "3456",
						}
					}
					return ci, nil
				},
			}, test.EnableContentTrust)
			cmd := NewPortCommand(cli)
			cmd.SetErr(ioutil.Discard)
			cmd.SetArgs([]string{"some_container", "80"})
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), tc.name+".golden")
		})
	}
}
