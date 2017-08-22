package container

import (
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	containerInspectFunc func(string) (types.ContainerJSON, error)
	execInspectFunc      func(ctx context.Context, execID string) (types.ContainerExecInspect, error)
}

func (cli *fakeClient) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	if cli.containerInspectFunc != nil {
		return cli.containerInspectFunc(containerID)
	}
	return types.ContainerJSON{}, nil
}

func (cli *fakeClient) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (cli *fakeClient) ContainerCommit(ctx context.Context, container string, options types.ContainerCommitOptions) (types.IDResponse, error) {
	return types.IDResponse{}, nil
}

func (cli *fakeClient) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
) (container.ContainerCreateCreatedBody, error) {
	return container.ContainerCreateCreatedBody{}, nil
}

func (cli *fakeClient) ContainerDiff(ctx context.Context, container string) ([]container.ContainerChangeResponseItem, error) {
	return nil, nil
}

func (cli *fakeClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecConfig) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (cli *fakeClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	return types.IDResponse{}, nil
}

func (cli *fakeClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	if cli.execInspectFunc != nil {
		return cli.execInspectFunc(ctx, execID)
	}
	return types.ContainerExecInspect{}, nil
}

func (cli *fakeClient) ContainerExecResize(ctx context.Context, execID string, options types.ResizeOptions) error {
	return nil
}

func (cli *fakeClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	return nil
}

func (cli *fakeClient) ContainerExport(ctx context.Context, container string) (io.ReadCloser, error) {
	return nil, nil
}

func (cli *fakeClient) ContainerInspectWithRaw(ctx context.Context, container string, getSize bool) (types.ContainerJSON, []byte, error) {
	return types.ContainerJSON{}, nil, nil
}

func (cli *fakeClient) ContainerKill(ctx context.Context, container, signal string) error {
	return nil
}

func (cli *fakeClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return nil, nil
}

func (cli *fakeClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	return nil, nil
}

func (cli *fakeClient) ContainerPause(ctx context.Context, container string) error {
	return nil
}

func (cli *fakeClient) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
	return nil
}

func (cli *fakeClient) ContainerRename(ctx context.Context, container, newContainerName string) error {
	return nil
}

func (cli *fakeClient) ContainerResize(ctx context.Context, container string, options types.ResizeOptions) error {
	return nil
}

func (cli *fakeClient) ContainerRestart(ctx context.Context, container string, timeout *time.Duration) error {
	return nil
}

func (cli *fakeClient) ContainerStatPath(ctx context.Context, container, path string) (types.ContainerPathStat, error) {
	return types.ContainerPathStat{}, nil
}

func (cli *fakeClient) ContainerStats(ctx context.Context, container string, stream bool) (types.ContainerStats, error) {
	return types.ContainerStats{}, nil
}

func (cli *fakeClient) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	return nil
}

func (cli *fakeClient) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
	return nil
}

func (cli *fakeClient) ContainerTop(ctx context.Context, containerID string, arguments []string) (container.ContainerTopOKBody, error) {
	return container.ContainerTopOKBody{}, nil
}

func (cli *fakeClient) ContainerUnpause(ctx context.Context, container string) error {
	return nil
}

func (cli *fakeClient) ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return container.ContainerUpdateOKBody{}, nil
}

func (cli *fakeClient) ContainerWait(ctx context.Context, container string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return nil, nil
}

func (cli *fakeClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	return nil, types.ContainerPathStat{}, nil
}

func (cli *fakeClient) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	return nil
}

func (cli *fakeClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
	return types.ContainersPruneReport{}, nil
}
