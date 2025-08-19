package container

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewPortCommandOutput(t *testing.T) {
	testCases := []struct {
		name string
		ips  []string
		port string
	}{
		{
			name: "container-port-ipv4",
			ips:  []string{"0.0.0.0"},
			port: "80",
		},
		{
			name: "container-port-ipv6",
			ips:  []string{"::"},
			port: "80",
		},
		{
			name: "container-port-ipv6-and-ipv4",
			ips:  []string{"::", "0.0.0.0"},
			port: "80",
		},
		{
			name: "container-port-ipv6-and-ipv4-443-udp",
			ips:  []string{"::", "0.0.0.0"},
			port: "443/udp",
		},
		{
			name: "container-port-all-ports",
			ips:  []string{"::", "0.0.0.0"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				inspectFunc: func(string) (container.InspectResponse, error) {
					ci := container.InspectResponse{NetworkSettings: &container.NetworkSettings{}}
					ci.NetworkSettings.Ports = container.PortMap{
						"80/tcp":  make([]container.PortBinding, len(tc.ips)),
						"443/tcp": make([]container.PortBinding, len(tc.ips)),
						"443/udp": make([]container.PortBinding, len(tc.ips)),
					}
					for i, ip := range tc.ips {
						ci.NetworkSettings.Ports["80/tcp"][i] = container.PortBinding{
							HostIP: ip, HostPort: "3456",
						}
						ci.NetworkSettings.Ports["443/tcp"][i] = container.PortBinding{
							HostIP: ip, HostPort: "4567",
						}
						ci.NetworkSettings.Ports["443/udp"][i] = container.PortBinding{
							HostIP: ip, HostPort: "5678",
						}
					}
					return ci, nil
				},
			})
			cmd := newPortCommand(cli)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"some_container", tc.port})
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), tc.name+".golden")
		})
	}
}
