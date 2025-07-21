package config

import (
	"context"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	configCreateFunc  func(context.Context, swarm.ConfigSpec) (swarm.ConfigCreateResponse, error)
	configInspectFunc func(context.Context, string) (swarm.Config, []byte, error)
	configListFunc    func(context.Context, swarm.ConfigListOptions) ([]swarm.Config, error)
	configRemoveFunc  func(string) error
}

func (c *fakeClient) ConfigCreate(ctx context.Context, spec swarm.ConfigSpec) (swarm.ConfigCreateResponse, error) {
	if c.configCreateFunc != nil {
		return c.configCreateFunc(ctx, spec)
	}
	return swarm.ConfigCreateResponse{}, nil
}

func (c *fakeClient) ConfigInspectWithRaw(ctx context.Context, id string) (swarm.Config, []byte, error) {
	if c.configInspectFunc != nil {
		return c.configInspectFunc(ctx, id)
	}
	return swarm.Config{}, nil, nil
}

func (c *fakeClient) ConfigList(ctx context.Context, options swarm.ConfigListOptions) ([]swarm.Config, error) {
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
