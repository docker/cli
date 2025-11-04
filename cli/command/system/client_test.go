package system

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client

	version            string
	containerListFunc  func(context.Context, client.ContainerListOptions) ([]container.Summary, error)
	containerPruneFunc func(ctx context.Context, options client.ContainerPruneOptions) (client.ContainerPruneResult, error)
	eventsFn           func(context.Context, client.EventsListOptions) (<-chan events.Message, <-chan error)
	imageListFunc      func(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error)
	infoFunc           func(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error)
	networkListFunc    func(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error)
	networkPruneFunc   func(ctx context.Context, options client.NetworkPruneOptions) (client.NetworkPruneResult, error)
	nodeListFunc       func(ctx context.Context, options client.NodeListOptions) (client.NodeListResult, error)
	serverVersion      func(ctx context.Context, options client.ServerVersionOptions) (client.ServerVersionResult, error)
	volumeListFunc     func(ctx context.Context, options client.VolumeListOptions) (client.VolumeListResult, error)
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	if cli.containerListFunc != nil {
		res, err := cli.containerListFunc(ctx, options)
		return client.ContainerListResult{
			Items: res,
		}, err
	}
	return client.ContainerListResult{}, nil
}

func (cli *fakeClient) ContainerPrune(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	if cli.containerPruneFunc != nil {
		return cli.containerPruneFunc(ctx, opts)
	}
	return client.ContainerPruneResult{}, nil
}

func (cli *fakeClient) Events(ctx context.Context, opts client.EventsListOptions) client.EventsResult {
	eventC, errC := cli.eventsFn(ctx, opts)
	return client.EventsResult{
		Messages: eventC,
		Err:      errC,
	}
}

func (cli *fakeClient) ImageList(ctx context.Context, options client.ImageListOptions) (client.ImageListResult, error) {
	if cli.imageListFunc != nil {
		return cli.imageListFunc(ctx, options)
	}
	return client.ImageListResult{}, nil
}

func (cli *fakeClient) Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc(ctx, options)
	}
	return client.SystemInfoResult{}, nil
}

func (cli *fakeClient) NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	if cli.networkListFunc != nil {
		return cli.networkListFunc(ctx, options)
	}
	return client.NetworkListResult{}, nil
}

func (cli *fakeClient) NetworksPrune(ctx context.Context, opts client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	if cli.networkPruneFunc != nil {
		return cli.networkPruneFunc(ctx, opts)
	}
	return client.NetworkPruneResult{}, nil
}

func (cli *fakeClient) NodeList(ctx context.Context, options client.NodeListOptions) (client.NodeListResult, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc(ctx, options)
	}
	return client.NodeListResult{}, nil
}

func (cli *fakeClient) ServerVersion(ctx context.Context, options client.ServerVersionOptions) (client.ServerVersionResult, error) {
	return cli.serverVersion(ctx, options)
}

func (cli *fakeClient) VolumeList(ctx context.Context, options client.VolumeListOptions) (client.VolumeListResult, error) {
	if cli.volumeListFunc != nil {
		return cli.volumeListFunc(ctx, options)
	}
	return client.VolumeListResult{}, nil
}
