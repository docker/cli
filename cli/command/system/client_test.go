package system

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version            string
	serverVersion      func(ctx context.Context) (types.Version, error)
	eventsFn           func(context.Context, types.EventsOptions) (<-chan events.Message, <-chan error)
	containerPruneFunc func(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error)
	networkPruneFunc   func(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error)
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

func (cli *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
	if cli.containerPruneFunc != nil {
		return cli.containerPruneFunc(ctx, pruneFilters)
	}
	return types.ContainersPruneReport{}, nil
}

func (cli *fakeClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error) {
	if cli.networkPruneFunc != nil {
		return cli.networkPruneFunc(ctx, pruneFilter)
	}
	return types.NetworksPruneReport{}, nil
}
