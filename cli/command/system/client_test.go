package system

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version            string
	containerListFunc  func(context.Context, container.ListOptions) ([]types.Container, error)
	containerPruneFunc func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	eventsFn           func(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	imageListFunc      func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	infoFunc           func(ctx context.Context) (system.Info, error)
	networkListFunc    func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	networkPruneFunc   func(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error)
	nodeListFunc       func(ctx context.Context, options types.NodeListOptions) ([]swarm.Node, error)
	serverVersion      func(ctx context.Context) (types.Version, error)
	volumeListFunc     func(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	if cli.containerListFunc != nil {
		return cli.containerListFunc(ctx, options)
	}
	return []types.Container{}, nil
}

func (cli *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
	if cli.containerPruneFunc != nil {
		return cli.containerPruneFunc(ctx, pruneFilters)
	}
	return container.PruneReport{}, nil
}

func (cli *fakeClient) Events(ctx context.Context, opts events.ListOptions) (<-chan events.Message, <-chan error) {
	return cli.eventsFn(ctx, opts)
}

func (cli *fakeClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	if cli.imageListFunc != nil {
		return cli.imageListFunc(ctx, options)
	}
	return []image.Summary{}, nil
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

func (cli *fakeClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (network.PruneReport, error) {
	if cli.networkPruneFunc != nil {
		return cli.networkPruneFunc(ctx, pruneFilter)
	}
	return network.PruneReport{}, nil
}

func (cli *fakeClient) NodeList(ctx context.Context, options types.NodeListOptions) ([]swarm.Node, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc(ctx, options)
	}
	return []swarm.Node{}, nil
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return cli.serverVersion(ctx)
}

func (cli *fakeClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	if cli.volumeListFunc != nil {
		return cli.volumeListFunc(ctx, options)
	}
	return volume.ListResponse{}, nil
}
