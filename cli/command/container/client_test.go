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
	inspectFunc         func(string) (container.InspectResponse, error)
	execInspectFunc     func(execID string) (container.ExecInspect, error)
	execCreateFunc      func(containerID string, options container.ExecOptions) (types.IDResponse, error)
	createContainerFunc func(config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		platform *specs.Platform,
		containerName string) (container.CreateResponse, error)
	containerStartFunc      func(containerID string, options container.StartOptions) error
	imageCreateFunc         func(parentReference string, options image.CreateOptions) (io.ReadCloser, error)
	infoFunc                func() (system.Info, error)
	containerStatPathFunc   func(containerID, path string) (container.PathStat, error)
	containerCopyFromFunc   func(containerID, srcPath string) (io.ReadCloser, container.PathStat, error)
	logFunc                 func(string, container.LogsOptions) (io.ReadCloser, error)
	waitFunc                func(string) (<-chan container.WaitResponse, <-chan error)
	containerListFunc       func(container.ListOptions) ([]container.Summary, error)
	containerExportFunc     func(string) (io.ReadCloser, error)
	containerExecResizeFunc func(id string, options container.ResizeOptions) error
	containerRemoveFunc     func(ctx context.Context, containerID string, options container.RemoveOptions) error
	containerRestartFunc    func(ctx context.Context, containerID string, options container.StopOptions) error
	containerStopFunc       func(ctx context.Context, containerID string, options container.StopOptions) error
	containerKillFunc       func(ctx context.Context, containerID, signal string) error
	containerPruneFunc      func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	containerAttachFunc     func(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error)
	containerDiffFunc       func(ctx context.Context, containerID string) ([]container.FilesystemChange, error)
	containerRenameFunc     func(ctx context.Context, oldName, newName string) error
	Version                 string
}

func (f *fakeClient) ContainerList(_ context.Context, options container.ListOptions) ([]container.Summary, error) {
	if f.containerListFunc != nil {
		return f.containerListFunc(options)
	}
	return []container.Summary{}, nil
}

func (f *fakeClient) ContainerInspect(_ context.Context, containerID string) (container.InspectResponse, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(containerID)
	}
	return container.InspectResponse{}, nil
}

func (f *fakeClient) ContainerExecCreate(_ context.Context, containerID string, config container.ExecOptions) (types.IDResponse, error) {
	if f.execCreateFunc != nil {
		return f.execCreateFunc(containerID, config)
	}
	return types.IDResponse{}, nil
}

func (f *fakeClient) ContainerExecInspect(_ context.Context, execID string) (container.ExecInspect, error) {
	if f.execInspectFunc != nil {
		return f.execInspectFunc(execID)
	}
	return container.ExecInspect{}, nil
}

func (f *fakeClient) ContainerExecStart(context.Context, string, container.ExecStartOptions) error {
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

func (f *fakeClient) ContainerStatPath(_ context.Context, containerID, path string) (container.PathStat, error) {
	if f.containerStatPathFunc != nil {
		return f.containerStatPathFunc(containerID, path)
	}
	return container.PathStat{}, nil
}

func (f *fakeClient) CopyFromContainer(_ context.Context, containerID, srcPath string) (io.ReadCloser, container.PathStat, error) {
	if f.containerCopyFromFunc != nil {
		return f.containerCopyFromFunc(containerID, srcPath)
	}
	return nil, container.PathStat{}, nil
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

func (f *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
	if f.containerPruneFunc != nil {
		return f.containerPruneFunc(ctx, pruneFilters)
	}
	return container.PruneReport{}, nil
}

func (f *fakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	if f.containerRestartFunc != nil {
		return f.containerRestartFunc(ctx, containerID, options)
	}
	return nil
}

func (f *fakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	if f.containerStopFunc != nil {
		return f.containerStopFunc(ctx, containerID, options)
	}
	return nil
}

func (f *fakeClient) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	if f.containerAttachFunc != nil {
		return f.containerAttachFunc(ctx, containerID, options)
	}
	return types.HijackedResponse{}, nil
}

func (f *fakeClient) ContainerDiff(ctx context.Context, containerID string) ([]container.FilesystemChange, error) {
	if f.containerDiffFunc != nil {
		return f.containerDiffFunc(ctx, containerID)
	}

	return []container.FilesystemChange{}, nil
}

func (f *fakeClient) ContainerRename(ctx context.Context, oldName, newName string) error {
	if f.containerRenameFunc != nil {
		return f.containerRenameFunc(ctx, oldName, newName)
	}

	return nil
}
