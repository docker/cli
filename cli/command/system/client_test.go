package system

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.APIClient

	version            string
	serverVersion      func(ctx context.Context) (types.Version, error)
	eventsFn           func(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	containerPruneFunc func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	networkPruneFunc   func(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error)
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return cli.serverVersion(ctx)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) Events(ctx context.Context, opts events.ListOptions) (<-chan events.Message, <-chan error) {
	return cli.eventsFn(ctx, opts)
}

func (cli *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
	if cli.containerPruneFunc != nil {
		return cli.containerPruneFunc(ctx, pruneFilters)
	}
	return container.PruneReport{}, nil
}

func (cli *fakeClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error) {
	if cli.networkPruneFunc != nil {
		return cli.networkPruneFunc(ctx, pruneFilter)
	}
	return network.PruneReport{}, nil
}
