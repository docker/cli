package network

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	networkCreateFunc     func(ctx context.Context, name string, options client.NetworkCreateOptions) (client.NetworkCreateResult, error)
	networkConnectFunc    func(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error)
	networkDisconnectFunc func(ctx context.Context, networkID string, options client.NetworkDisconnectOptions) (client.NetworkDisconnectResult, error)
	networkRemoveFunc     func(ctx context.Context, networkID string) error
	networkListFunc       func(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error)
	networkPruneFunc      func(ctx context.Context, options client.NetworkPruneOptions) (client.NetworkPruneResult, error)
	networkInspectFunc    func(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error)
}

func (c *fakeClient) NetworkCreate(ctx context.Context, name string, options client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
	if c.networkCreateFunc != nil {
		return c.networkCreateFunc(ctx, name, options)
	}
	return client.NetworkCreateResult{}, nil
}

func (c *fakeClient) NetworkConnect(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	if c.networkConnectFunc != nil {
		return c.networkConnectFunc(ctx, networkID, options)
	}
	return client.NetworkConnectResult{}, nil
}

func (c *fakeClient) NetworkDisconnect(ctx context.Context, networkID string, options client.NetworkDisconnectOptions) (client.NetworkDisconnectResult, error) {
	if c.networkDisconnectFunc != nil {
		return c.networkDisconnectFunc(ctx, networkID, options)
	}
	return client.NetworkDisconnectResult{}, nil
}

func (c *fakeClient) NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	if c.networkListFunc != nil {
		return c.networkListFunc(ctx, options)
	}
	return client.NetworkListResult{}, nil
}

func (c *fakeClient) NetworkRemove(ctx context.Context, networkID string, _ client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	if c.networkRemoveFunc != nil {
		return client.NetworkRemoveResult{}, c.networkRemoveFunc(ctx, networkID)
	}
	return client.NetworkRemoveResult{}, nil
}

func (c *fakeClient) NetworkInspect(ctx context.Context, networkID string, opts client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	if c.networkInspectFunc != nil {
		return c.networkInspectFunc(ctx, networkID, opts)
	}
	return client.NetworkInspectResult{}, nil
}

func (c *fakeClient) NetworksPrune(ctx context.Context, opts client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	if c.networkPruneFunc != nil {
		return c.networkPruneFunc(ctx, opts)
	}
	return client.NetworkPruneResult{}, nil
}
