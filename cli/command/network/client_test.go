package network

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	networkCreateFunc func(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
}

func (c *fakeClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	if c.networkCreateFunc != nil {
		return c.networkCreateFunc(ctx, name, options)
	}
	return types.NetworkCreateResponse{}, nil
}
