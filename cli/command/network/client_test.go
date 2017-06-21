package network

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"github.com/docker/docker/api/types/filters"
)

type fakeClient struct {
	client.Client
	networkCreateFunc     func(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	networkConnectFunc    func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
	networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
	NetworkInspectFunc func(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)
	NetworkInspectWithRawFunc func(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, []byte, error)
	NetworkListFunc func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	NetworkRemoveFunc func(ctx context.Context, networkID string) error
	NetworksPruneFunc func(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error)
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
