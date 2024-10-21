package system

import (
	"context"

	"github.com/docker/docker/api/types/system"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version            string
	serverVersion      func(ctx context.Context) (types.Version, error)
	eventsFn           func(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	containerPruneFunc func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	networkPruneFunc   func(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error)
	containerListFunc  func(context.Context, container.ListOptions) ([]container.Summary, error)
	infoFunc           func(ctx context.Context) (system.Info, error)
	networkListFunc    func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
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

func (cli *fakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	if cli.containerListFunc != nil {
		return cli.containerListFunc(ctx, options)
	}
	return []container.Summary{}, nil
}

func (cli *fakeClient) Info(ctx context.Context) (system.Info, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc(ctx)
	}
	return system.Info{}, nil
}

func (cli *fakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	if cli.networkListFunc != nil {
		return cli.networkListFunc(ctx, options)
	}
	return []network.Summary{}, nil
}
