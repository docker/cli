package secret

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	secretCreateFunc  func(context.Context, swarm.SecretSpec) (types.SecretCreateResponse, error)
	secretInspectFunc func(context.Context, string) (swarm.Secret, []byte, error)
	secretListFunc    func(context.Context, types.SecretListOptions) ([]swarm.Secret, error)
	secretRemoveFunc  func(context.Context, string) error
}

func (c *fakeClient) SecretCreate(ctx context.Context, spec swarm.SecretSpec) (types.SecretCreateResponse, error) {
	if c.secretCreateFunc != nil {
		return c.secretCreateFunc(ctx, spec)
	}
	return types.SecretCreateResponse{}, nil
}

func (c *fakeClient) SecretInspectWithRaw(ctx context.Context, id string) (swarm.Secret, []byte, error) {
	if c.secretInspectFunc != nil {
		return c.secretInspectFunc(ctx, id)
	}
	return swarm.Secret{}, nil, nil
}

func (c *fakeClient) SecretList(ctx context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
	if c.secretListFunc != nil {
		return c.secretListFunc(ctx, options)
	}
	return []swarm.Secret{}, nil
}

func (c *fakeClient) SecretRemove(ctx context.Context, name string) error {
	if c.secretRemoveFunc != nil {
		return c.secretRemoveFunc(ctx, name)
	}
	return nil
}
