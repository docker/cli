package network

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	networkCreateFunc     func(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	networkConnectFunc    func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
	networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
	networkListFunc       func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
}

func (c *fakeClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	if c.networkCreateFunc != nil {
		return c.networkCreateFunc(ctx, name, options)
	}
	return types.NetworkCreateResponse{}, nil
}

func (c *fakeClient) NetworkConnect(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
	if c.networkConnectFunc != nil {
		return c.networkConnectFunc(ctx, networkID, container, config)
	}
	return nil
}

func (c *fakeClient) NetworkDisconnect(ctx context.Context, networkID, container string, force bool) error {
	if c.networkDisconnectFunc != nil {
		return c.networkDisconnectFunc(ctx, networkID, container, force)
	}
	return nil
}

func (c *fakeClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	if c.networkListFunc != nil {
		return c.networkListFunc(ctx, options)
	}
	return []types.NetworkResource{}, nil
}
