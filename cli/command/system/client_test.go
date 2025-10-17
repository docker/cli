package system

import (
	"context"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client

	version            string
	containerListFunc  func(context.Context, client.ContainerListOptions) ([]container.Summary, error)
	containerPruneFunc func(ctx context.Context, options client.ContainerPruneOptions) (client.ContainerPruneResult, error)
	eventsFn           func(context.Context, client.EventsListOptions) (<-chan events.Message, <-chan error)
	imageListFunc      func(ctx context.Context, options client.ImageListOptions) ([]image.Summary, error)
	infoFunc           func(ctx context.Context) (system.Info, error)
	networkListFunc    func(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error)
	networkPruneFunc   func(ctx context.Context, options client.NetworkPruneOptions) (client.NetworkPruneResult, error)
	nodeListFunc       func(ctx context.Context, options client.NodeListOptions) ([]swarm.Node, error)
	serverVersion      func(ctx context.Context) (types.Version, error)
	volumeListFunc     func(ctx context.Context, options client.VolumeListOptions) (volume.ListResponse, error)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) ContainerList(ctx context.Context, options client.ContainerListOptions) ([]container.Summary, error) {
	if cli.containerListFunc != nil {
		return cli.containerListFunc(ctx, options)
	}
	return []container.Summary{}, nil
}

func (cli *fakeClient) ContainersPrune(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	if cli.containerPruneFunc != nil {
		return cli.containerPruneFunc(ctx, opts)
	}
	return client.ContainerPruneResult{}, nil
}

func (cli *fakeClient) Events(ctx context.Context, opts client.EventsListOptions) (<-chan events.Message, <-chan error) {
	return cli.eventsFn(ctx, opts)
}

func (cli *fakeClient) ImageList(ctx context.Context, options client.ImageListOptions) ([]image.Summary, error) {
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

func (cli *fakeClient) NetworkList(ctx context.Context, options client.NetworkListOptions) ([]network.Summary, error) {
	if cli.networkListFunc != nil {
		return cli.networkListFunc(ctx, options)
	}
	return []network.Summary{}, nil
}

func (cli *fakeClient) NetworksPrune(ctx context.Context, opts client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	if cli.networkPruneFunc != nil {
		return cli.networkPruneFunc(ctx, opts)
	}
	return client.NetworkPruneResult{}, nil
}

func (cli *fakeClient) NodeList(ctx context.Context, options client.NodeListOptions) ([]swarm.Node, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc(ctx, options)
	}
	return []swarm.Node{}, nil
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return cli.serverVersion(ctx)
}

func (cli *fakeClient) VolumeList(ctx context.Context, options client.VolumeListOptions) (volume.ListResponse, error) {
	if cli.volumeListFunc != nil {
		return cli.volumeListFunc(ctx, options)
	}
	return volume.ListResponse{}, nil
}
