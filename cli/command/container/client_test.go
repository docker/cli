package container

import (
	"context"
	"io"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	inspectFunc             func(string) (client.ContainerInspectResult, error)
	execInspectFunc         func(execID string) (client.ExecInspectResult, error)
	execCreateFunc          func(containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error)
	createContainerFunc     func(options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	containerStartFunc      func(containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	imageCreateFunc         func(ctx context.Context, parentReference string, options client.ImageCreateOptions) (client.ImageCreateResult, error)
	infoFunc                func() (system.Info, error)
	containerStatPathFunc   func(containerID, path string) (container.PathStat, error)
	containerCopyFromFunc   func(containerID, srcPath string) (io.ReadCloser, container.PathStat, error)
	logFunc                 func(string, client.ContainerLogsOptions) (io.ReadCloser, error)
	waitFunc                func(string) (<-chan container.WaitResponse, <-chan error)
	containerListFunc       func(client.ContainerListOptions) ([]container.Summary, error)
	containerExportFunc     func(string) (io.ReadCloser, error)
	containerExecResizeFunc func(id string, options client.ExecResizeOptions) (client.ExecResizeResult, error)
	containerRemoveFunc     func(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	containerRestartFunc    func(ctx context.Context, containerID string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error)
	containerStopFunc       func(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	containerKillFunc       func(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error)
	containerPruneFunc      func(ctx context.Context, options client.ContainerPruneOptions) (client.ContainerPruneResult, error)
	containerAttachFunc     func(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error)
	containerDiffFunc       func(ctx context.Context, containerID string) (client.ContainerDiffResult, error)
	containerRenameFunc     func(ctx context.Context, oldName, newName string) error
	containerCommitFunc     func(ctx context.Context, container string, options client.ContainerCommitOptions) (client.ContainerCommitResult, error)
	containerPauseFunc      func(ctx context.Context, container string, options client.ContainerPauseOptions) (client.ContainerPauseResult, error)
	Version                 string
}

func (f *fakeClient) ContainerList(_ context.Context, options client.ContainerListOptions) ([]container.Summary, error) {
	if f.containerListFunc != nil {
		return f.containerListFunc(options)
	}
	return []container.Summary{}, nil
}

func (f *fakeClient) ContainerInspect(_ context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(containerID)
	}
	return client.ContainerInspectResult{}, nil
}

func (f *fakeClient) ExecCreate(_ context.Context, containerID string, config client.ExecCreateOptions) (client.ExecCreateResult, error) {
	if f.execCreateFunc != nil {
		return f.execCreateFunc(containerID, config)
	}
	return client.ExecCreateResult{}, nil
}

func (f *fakeClient) ExecInspect(_ context.Context, execID string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
	if f.execInspectFunc != nil {
		return f.execInspectFunc(execID)
	}
	return client.ExecInspectResult{}, nil
}

func (*fakeClient) ExecStart(context.Context, string, client.ExecStartOptions) (client.ExecStartResult, error) {
	return client.ExecStartResult{}, nil
}

func (f *fakeClient) ContainerCreate(_ context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	if f.createContainerFunc != nil {
		return f.createContainerFunc(options)
	}
	return client.ContainerCreateResult{}, nil
}

func (f *fakeClient) ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	if f.containerRemoveFunc != nil {
		return f.containerRemoveFunc(ctx, containerID, options)
	}
	return client.ContainerRemoveResult{}, nil
}

func (f *fakeClient) ImageCreate(ctx context.Context, parentReference string, options client.ImageCreateOptions) (client.ImageCreateResult, error) {
	if f.imageCreateFunc != nil {
		return f.imageCreateFunc(ctx, parentReference, options)
	}
	return client.ImageCreateResult{}, nil
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

func (f *fakeClient) ContainerLogs(_ context.Context, containerID string, options client.ContainerLogsOptions) (io.ReadCloser, error) {
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

func (f *fakeClient) ContainerStart(_ context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
	if f.containerStartFunc != nil {
		return f.containerStartFunc(containerID, options)
	}
	return client.ContainerStartResult{}, nil
}

func (f *fakeClient) ContainerExport(_ context.Context, containerID string) (io.ReadCloser, error) {
	if f.containerExportFunc != nil {
		return f.containerExportFunc(containerID)
	}
	return nil, nil
}

func (f *fakeClient) ExecResize(_ context.Context, id string, options client.ExecResizeOptions) (client.ExecResizeResult, error) {
	if f.containerExecResizeFunc != nil {
		return f.containerExecResizeFunc(id, options)
	}
	return client.ExecResizeResult{}, nil
}

func (f *fakeClient) ContainerKill(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
	if f.containerKillFunc != nil {
		return f.containerKillFunc(ctx, containerID, options)
	}
	return client.ContainerKillResult{}, nil
}

func (f *fakeClient) ContainersPrune(ctx context.Context, options client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	if f.containerPruneFunc != nil {
		return f.containerPruneFunc(ctx, options)
	}
	return client.ContainerPruneResult{}, nil
}

func (f *fakeClient) ContainerRestart(ctx context.Context, containerID string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
	if f.containerRestartFunc != nil {
		return f.containerRestartFunc(ctx, containerID, options)
	}
	return client.ContainerRestartResult{}, nil
}

func (f *fakeClient) ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	if f.containerStopFunc != nil {
		return f.containerStopFunc(ctx, containerID, options)
	}
	return client.ContainerStopResult{}, nil
}

func (f *fakeClient) ContainerAttach(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
	if f.containerAttachFunc != nil {
		return f.containerAttachFunc(ctx, containerID, options)
	}
	return client.ContainerAttachResult{}, nil
}

func (f *fakeClient) ContainerDiff(ctx context.Context, containerID string, _ client.ContainerDiffOptions) (client.ContainerDiffResult, error) {
	if f.containerDiffFunc != nil {
		return f.containerDiffFunc(ctx, containerID)
	}

	return client.ContainerDiffResult{}, nil
}

func (f *fakeClient) ContainerRename(ctx context.Context, oldName, newName string) error {
	if f.containerRenameFunc != nil {
		return f.containerRenameFunc(ctx, oldName, newName)
	}

	return nil
}

func (f *fakeClient) ContainerCommit(ctx context.Context, containerID string, options client.ContainerCommitOptions) (client.ContainerCommitResult, error) {
	if f.containerCommitFunc != nil {
		return f.containerCommitFunc(ctx, containerID, options)
	}
	return client.ContainerCommitResult{}, nil
}

func (f *fakeClient) ContainerPause(ctx context.Context, containerID string, options client.ContainerPauseOptions) (client.ContainerPauseResult, error) {
	if f.containerPauseFunc != nil {
		return f.containerPauseFunc(ctx, containerID, options)
	}

	return client.ContainerPauseResult{}, nil
}
