package builder

import (
	"context"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	builderPruneFunc func(ctx context.Context, opts build.CachePruneOptions) (*build.CachePruneReport, error)
}

func (c *fakeClient) BuildCachePrune(ctx context.Context, opts build.CachePruneOptions) (*build.CachePruneReport, error) {
	if c.builderPruneFunc != nil {
		return c.builderPruneFunc(ctx, opts)
	}
	return nil, nil
}
