package network

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// FakeClient is a fake NetworkAPIClient
type FakeClient struct {
	client.NetworkAPIClient
	NetworkInspectFunc func(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)
}

// NetworkInspect fakes inspecting a network
func (c *FakeClient) NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	if c.NetworkInspectFunc != nil {
		return c.NetworkInspectFunc(ctx, networkID, options)
	}
	return types.NetworkResource{}, nil
}
