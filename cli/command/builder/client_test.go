package builder

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	builderPruneFunc func(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error)
}

func (c *fakeClient) BuildCachePrune(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
	if c.builderPruneFunc != nil {
		return c.builderPruneFunc(ctx, opts)
	}
	return nil, nil
}
