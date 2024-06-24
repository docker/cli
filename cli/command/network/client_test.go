package network

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	networkCreateFunc     func(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
	networkConnectFunc    func(ctx context.Context, networkID, container string, config *network.EndpointSettings) error
	networkDisconnectFunc func(ctx context.Context, networkID, container string, force bool) error
	networkRemoveFunc     func(ctx context.Context, networkID string) error
	networkListFunc       func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	networkPruneFunc      func(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error)
	networkInspectFunc    func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, []byte, error)
}

func (c *fakeClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	if c.networkCreateFunc != nil {
		return c.networkCreateFunc(ctx, name, options)
	}
	return network.CreateResponse{}, nil
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

func (c *fakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	if c.networkListFunc != nil {
		return c.networkListFunc(ctx, options)
	}
	return []network.Inspect{}, nil
}

func (c *fakeClient) NetworkRemove(ctx context.Context, networkID string) error {
	if c.networkRemoveFunc != nil {
		return c.networkRemoveFunc(ctx, networkID)
	}
	return nil
}

func (c *fakeClient) NetworkInspectWithRaw(ctx context.Context, networkID string, opts network.InspectOptions) (network.Inspect, []byte, error) {
	if c.networkInspectFunc != nil {
		return c.networkInspectFunc(ctx, networkID, opts)
	}
	return network.Inspect{}, nil, nil
}

func (c *fakeClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error) {
	if c.networkPruneFunc != nil {
		return c.networkPruneFunc(ctx, pruneFilter)
	}
	return network.PruneReport{}, nil
}
