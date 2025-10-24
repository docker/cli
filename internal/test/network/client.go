package network

import (
	"context"

	"github.com/moby/moby/client"
)

// FakeClient is a fake NetworkAPIClient
type FakeClient struct {
	client.NetworkAPIClient
	NetworkInspectFunc func(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error)
}

// NetworkInspect fakes inspecting a network
func (c *FakeClient) NetworkInspect(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	if c.NetworkInspectFunc != nil {
		return c.NetworkInspectFunc(ctx, networkID, options)
	}
	return client.NetworkInspectResult{}, nil
}
