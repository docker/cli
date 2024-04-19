package container

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

type fakeClient struct {
	client.Client
	inspectFunc         func(string) (types.ContainerJSON, error)
	execInspectFunc     func(execID string) (types.ContainerExecInspect, error)
	execCreateFunc      func(containerID string, config types.ExecConfig) (types.IDResponse, error)
	createContainerFunc func(config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		platform *specs.Platform,
		containerName string) (container.CreateResponse, error)
	containerStartFunc      func(containerID string, options container.StartOptions) error
	imageCreateFunc         func(parentReference string, options image.CreateOptions) (io.ReadCloser, error)
	infoFunc                func() (system.Info, error)
	containerStatPathFunc   func(containerID, path string) (types.ContainerPathStat, error)
	containerCopyFromFunc   func(containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)
	logFunc                 func(string, container.LogsOptions) (io.ReadCloser, error)
	waitFunc                func(string) (<-chan container.WaitResponse, <-chan error)
	containerListFunc       func(container.ListOptions) ([]types.Container, error)
	containerExportFunc     func(string) (io.ReadCloser, error)
	containerExecResizeFunc func(id string, options container.ResizeOptions) error
	containerRemoveFunc     func(ctx context.Context, containerID string, options container.RemoveOptions) error
	containerKillFunc       func(ctx context.Context, containerID, signal string) error
	containerPruneFunc      func(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error)
	Version                 string
}

func (f *fakeClient) ContainerList(_ context.Context, options container.ListOptions) ([]types.Container, error) {
	if f.containerListFunc != nil {
		return f.containerListFunc(options)
	}
	return []types.Container{}, nil
}

func (f *fakeClient) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(containerID)
	}
	return types.ContainerJSON{}, nil
}

func (f *fakeClient) ContainerExecCreate(_ context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error) {
	if f.execCreateFunc != nil {
		return f.execCreateFunc(containerID, config)
	}
	return types.IDResponse{}, nil
}

func (f *fakeClient) ContainerExecInspect(_ context.Context, execID string) (types.ContainerExecInspect, error) {
	if f.execInspectFunc != nil {
		return f.execInspectFunc(execID)
	}
	return types.ContainerExecInspect{}, nil
}

func (f *fakeClient) ContainerExecStart(context.Context, string, types.ExecStartCheck) error {
	return nil
}

func (f *fakeClient) ContainerCreate(
	_ context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *specs.Platform,
	containerName string,
) (container.CreateResponse, error) {
	if f.createContainerFunc != nil {
		return f.createContainerFunc(config, hostConfig, networkingConfig, platform, containerName)
	}
	return container.CreateResponse{}, nil
}

func (f *fakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if f.containerRemoveFunc != nil {
		return f.containerRemoveFunc(ctx, containerID, options)
	}
	return nil
}

func (f *fakeClient) ImageCreate(_ context.Context, parentReference string, options image.CreateOptions) (io.ReadCloser, error) {
	if f.imageCreateFunc != nil {
		return f.imageCreateFunc(parentReference, options)
	}
	return nil, nil
}

func (f *fakeClient) Info(_ context.Context) (system.Info, error) {
	if f.infoFunc != nil {
		return f.infoFunc()
	}
	return system.Info{}, nil
}

func (f *fakeClient) ContainerStatPath(_ context.Context, containerID, path string) (types.ContainerPathStat, error) {
	if f.containerStatPathFunc != nil {
		return f.containerStatPathFunc(containerID, path)
	}
	return types.ContainerPathStat{}, nil
}

func (f *fakeClient) CopyFromContainer(_ context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	if f.containerCopyFromFunc != nil {
		return f.containerCopyFromFunc(containerID, srcPath)
	}
	return nil, types.ContainerPathStat{}, nil
}

func (f *fakeClient) ContainerLogs(_ context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	if f.logFunc != nil {
		return f.logFunc(containerID, options)
	}
	return nil, nil
}

func (f *fakeClient) ClientVersion() string {
	return f.Version
}

func (f *fakeClient) ContainerWait(_ context.Context, containerID string, _ container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	if f.waitFunc != nil {
		return f.waitFunc(containerID)
	}
	return nil, nil
}

func (f *fakeClient) ContainerStart(_ context.Context, containerID string, options container.StartOptions) error {
	if f.containerStartFunc != nil {
		return f.containerStartFunc(containerID, options)
	}
	return nil
}

func (f *fakeClient) ContainerExport(_ context.Context, containerID string) (io.ReadCloser, error) {
	if f.containerExportFunc != nil {
		return f.containerExportFunc(containerID)
	}
	return nil, nil
}

func (f *fakeClient) ContainerExecResize(_ context.Context, id string, options container.ResizeOptions) error {
	if f.containerExecResizeFunc != nil {
		return f.containerExecResizeFunc(id, options)
	}
	return nil
}

func (f *fakeClient) ContainerKill(ctx context.Context, containerID, signal string) error {
	if f.containerKillFunc != nil {
		return f.containerKillFunc(ctx, containerID, signal)
	}
	return nil
}

func (f *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
	if f.containerPruneFunc != nil {
		return f.containerPruneFunc(ctx, pruneFilters)
	}
	return types.ContainersPruneReport{}, nil
}
