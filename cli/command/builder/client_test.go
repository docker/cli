package builder

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	builderPruneFunc func(ctx context.Context, opts client.BuildCachePruneOptions) (client.BuildCachePruneResult, error)
}

func (c *fakeClient) BuildCachePrune(ctx context.Context, opts client.BuildCachePruneOptions) (client.BuildCachePruneResult, error) {
	if c.builderPruneFunc != nil {
		return c.builderPruneFunc(ctx, opts)
	}
	return client.BuildCachePruneResult{}, nil
}
