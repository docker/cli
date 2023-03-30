package config

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	configCreateFunc  func(context.Context, swarm.ConfigSpec) (types.ConfigCreateResponse, error)
	configInspectFunc func(context.Context, string) (swarm.Config, []byte, error)
	configListFunc    func(context.Context, types.ConfigListOptions) ([]swarm.Config, error)
	configRemoveFunc  func(string) error
}

func (c *fakeClient) ConfigCreate(ctx context.Context, spec swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
	if c.configCreateFunc != nil {
		return c.configCreateFunc(ctx, spec)
	}
	return types.ConfigCreateResponse{}, nil
}

func (c *fakeClient) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	if c.configInspectFunc != nil {
		return c.configInspectFunc(ctx, id)
	}
	return swarm.Config{}, nil, nil
}

func (c *fakeClient) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	if c.configListFunc != nil {
		return c.configListFunc(ctx, options)
	}
	return []swarm.Config{}, nil
}

func (c *fakeClient) ConfigRemove(_ context.Context, name string) error {
	if c.configRemoveFunc != nil {
		return c.configRemoveFunc(name)
	}
	return nil
}
