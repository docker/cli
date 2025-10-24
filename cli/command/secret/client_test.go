package secret

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	secretCreateFunc  func(context.Context, client.SecretCreateOptions) (client.SecretCreateResult, error)
	secretInspectFunc func(context.Context, string, client.SecretInspectOptions) (client.SecretInspectResult, error)
	secretListFunc    func(context.Context, client.SecretListOptions) (client.SecretListResult, error)
	secretRemoveFunc  func(context.Context, string, client.SecretRemoveOptions) (client.SecretRemoveResult, error)
}

func (c *fakeClient) SecretCreate(ctx context.Context, options client.SecretCreateOptions) (client.SecretCreateResult, error) {
	if c.secretCreateFunc != nil {
		return c.secretCreateFunc(ctx, options)
	}
	return client.SecretCreateResult{}, nil
}

func (c *fakeClient) SecretInspect(ctx context.Context, id string, options client.SecretInspectOptions) (client.SecretInspectResult, error) {
	if c.secretInspectFunc != nil {
		return c.secretInspectFunc(ctx, id, options)
	}
	return client.SecretInspectResult{}, nil
}

func (c *fakeClient) SecretList(ctx context.Context, options client.SecretListOptions) (client.SecretListResult, error) {
	if c.secretListFunc != nil {
		return c.secretListFunc(ctx, options)
	}
	return client.SecretListResult{}, nil
}

func (c *fakeClient) SecretRemove(ctx context.Context, name string, options client.SecretRemoveOptions) (client.SecretRemoveResult, error) {
	if c.secretRemoveFunc != nil {
		return c.secretRemoveFunc(ctx, name, options)
	}
	return client.SecretRemoveResult{}, nil
}
