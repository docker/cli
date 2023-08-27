package system

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version       string
	serverVersion func(ctx context.Context) (types.Version, error)
	eventsFn      func(context.Context, types.EventsOptions) (<-chan events.Message, <-chan error)
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return cli.serverVersion(ctx)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) Events(ctx context.Context, opts types.EventsOptions) (<-chan events.Message, <-chan error) {
	return cli.eventsFn(ctx, opts)
}
