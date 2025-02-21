package network // import "docker.com/cli/v28/internal/test/network"

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// FakeClient is a fake NetworkAPIClient
type FakeClient struct {
	client.NetworkAPIClient
	NetworkInspectFunc func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error)
}

// NetworkInspect fakes inspecting a network
func (c *FakeClient) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	if c.NetworkInspectFunc != nil {
		return c.NetworkInspectFunc(ctx, networkID, options)
	}
	return network.Inspect{}, nil
}
