package config

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	configCreateFunc  func(context.Context, client.ConfigCreateOptions) (client.ConfigCreateResult, error)
	configInspectFunc func(context.Context, string, client.ConfigInspectOptions) (client.ConfigInspectResult, error)
	configListFunc    func(context.Context, client.ConfigListOptions) (client.ConfigListResult, error)
	configRemoveFunc  func(context.Context, string, client.ConfigRemoveOptions) (client.ConfigRemoveResult, error)
}

func (c *fakeClient) ConfigCreate(ctx context.Context, options client.ConfigCreateOptions) (client.ConfigCreateResult, error) {
	if c.configCreateFunc != nil {
		return c.configCreateFunc(ctx, options)
	}
	return client.ConfigCreateResult{}, nil
}

func (c *fakeClient) ConfigInspect(ctx context.Context, id string, options client.ConfigInspectOptions) (client.ConfigInspectResult, error) {
	if c.configInspectFunc != nil {
		return c.configInspectFunc(ctx, id, options)
	}
	return client.ConfigInspectResult{}, nil
}

func (c *fakeClient) ConfigList(ctx context.Context, options client.ConfigListOptions) (client.ConfigListResult, error) {
	if c.configListFunc != nil {
		return c.configListFunc(ctx, options)
	}
	return client.ConfigListResult{}, nil
}

func (c *fakeClient) ConfigRemove(ctx context.Context, name string, options client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
	if c.configRemoveFunc != nil {
		return c.configRemoveFunc(ctx, name, options)
	}
	return client.ConfigRemoveResult{}, nil
}
