package container

import (
	"io"
	"net/netip"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewPortCommandOutput(t *testing.T) {
	testCases := []struct {
		name string
		ips  []netip.Addr
		port string
	}{
		{
			name: "container-port-ipv4",
			ips:  []netip.Addr{netip.MustParseAddr("0.0.0.0")},
			port: "80",
		},
		{
			name: "container-port-ipv6",
			ips:  []netip.Addr{netip.MustParseAddr("::")},
			port: "80",
		},
		{
			name: "container-port-ipv6-and-ipv4",
			ips:  []netip.Addr{netip.MustParseAddr("::"), netip.MustParseAddr("0.0.0.0")},
			port: "80",
		},
		{
			name: "container-port-ipv6-and-ipv4-443-udp",
			ips:  []netip.Addr{netip.MustParseAddr("::"), netip.MustParseAddr("0.0.0.0")},
			port: "443/udp",
		},
		{
			name: "container-port-all-ports",
			ips:  []netip.Addr{netip.MustParseAddr("::"), netip.MustParseAddr("0.0.0.0")},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				inspectFunc: func(string) (client.ContainerInspectResult, error) {
					ci := container.InspectResponse{NetworkSettings: &container.NetworkSettings{}}
					ci.NetworkSettings.Ports = network.PortMap{
						network.MustParsePort("80/tcp"):  make([]network.PortBinding, len(tc.ips)),
						network.MustParsePort("443/tcp"): make([]network.PortBinding, len(tc.ips)),
						network.MustParsePort("443/udp"): make([]network.PortBinding, len(tc.ips)),
					}
					for i, ip := range tc.ips {
						ci.NetworkSettings.Ports[network.MustParsePort("80/tcp")][i] = network.PortBinding{
							HostIP: ip, HostPort: "3456",
						}
						ci.NetworkSettings.Ports[network.MustParsePort("443/tcp")][i] = network.PortBinding{
							HostIP: ip, HostPort: "4567",
						}
						ci.NetworkSettings.Ports[network.MustParsePort("443/udp")][i] = network.PortBinding{
							HostIP: ip, HostPort: "5678",
						}
					}
					return client.ContainerInspectResult{Container: ci}, nil
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
