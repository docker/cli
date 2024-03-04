package network

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	networkCreateFunc     func(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	networkConnectFunc    func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
	networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
	networkRemoveFunc     func(ctx context.Context, networkID string) error
	networkListFunc       func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	networkPruneFunc      func(ctx context.Context, pruneFilters filters.Args) (types.NetworksPruneReport, error)
	networkInspectFunc    func(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, []byte, error)
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

func (c *fakeClient) NetworkRemove(ctx context.Context, networkID string) error {
	if c.networkRemoveFunc != nil {
		return c.networkRemoveFunc(ctx, networkID)
	}
	return nil
}

func (c *fakeClient) NetworkInspectWithRaw(ctx context.Context, networkID string, opts types.NetworkInspectOptions) (types.NetworkResource, []byte, error) {
	if c.networkInspectFunc != nil {
		return c.networkInspectFunc(ctx, networkID, opts)
	}
	return types.NetworkResource{}, nil, nil
}

func (c *fakeClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error) {
	if c.networkPruneFunc != nil {
		return c.networkPruneFunc(ctx, pruneFilter)
	}
	return types.NetworksPruneReport{}, nil
}
