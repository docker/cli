//go:build linux

package standalone

import (
	"context"
	"fmt"
	"net/netip"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func (c *apiClient) NetworkList(_ context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	res := client.NetworkListResult{Items: []network.Summary{}}
	for _, name := range builtinNetworks {
		sum := c.eng.networkSummary(name)
		if !matchNetworkFilters(options.Filters, sum) {
			continue
		}
		res.Items = append(res.Items, sum)
	}
	return res, nil
}

func matchNetworkFilters(f client.Filters, sum network.Summary) bool {
	for key, values := range f {
		switch key {
		case "name":
			if _, ok := values[sum.Name]; !ok {
				return false
			}
		case "driver":
			if _, ok := values[sum.Driver]; !ok {
				return false
			}
		case "type":
			if _, ok := values["builtin"]; !ok {
				return false
			}
		case "scope":
			if _, ok := values["local"]; !ok {
				return false
			}
		}
	}
	return true
}

func (c *apiClient) NetworkInspect(ctx context.Context, name string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	var found bool
	for _, n := range builtinNetworks {
		if n == name || "standalone-"+n == name {
			name, found = n, true
		}
	}
	if !found {
		return client.NetworkInspectResult{}, fmt.Errorf("network %s not found: %w", name, cerrdefs.ErrNotFound)
	}
	sum := c.eng.networkSummary(name)
	insp := network.Inspect{Network: sum.Network, Containers: map[string]network.EndpointResource{}}
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ctrs, err := s.containers().List(ctx)
		if err != nil {
			return err
		}
		for _, ctr := range ctrs {
			m, err := unmarshalMeta(ctr)
			if err != nil || m.State.Status != statusRunning {
				continue
			}
			if networkModeName(m.HostConfig) != name {
				continue
			}
			ep := network.EndpointResource{Name: m.Name, EndpointID: ctr.ID}
			if ip, err := netip.ParseAddr(m.Network.IPAddress); err == nil {
				ep.IPv4Address = netip.PrefixFrom(ip, m.Network.IPPrefix)
			}
			insp.Containers[ctr.ID] = ep
		}
		return nil
	})
	return client.NetworkInspectResult{Network: insp}, err
}

func (*apiClient) NetworkCreate(context.Context, string, client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
	return client.NetworkCreateResult{}, fmt.Errorf("user-defined networks are not supported in standalone mode; use --network bridge|host|none: %w", cerrdefs.ErrNotImplemented)
}

func (*apiClient) NetworkRemove(context.Context, string, client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	return client.NetworkRemoveResult{}, fmt.Errorf("built-in networks cannot be removed: %w", cerrdefs.ErrInvalidArgument)
}

func (*apiClient) NetworkPrune(context.Context, client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	return client.NetworkPruneResult{}, nil
}
