package container

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	inspectFunc     func(string) (types.ContainerJSON, error)
	execInspectFunc func(execID string) (types.ContainerExecInspect, error)
	execCreateFunc  func(container string, config types.ExecConfig) (types.IDResponse, error)
}

func (f *fakeClient) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(containerID)
	}
	return types.ContainerJSON{}, nil
}

func (f *fakeClient) ContainerExecCreate(_ context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	if f.execCreateFunc != nil {
		return f.execCreateFunc(container, config)
	}
	return types.IDResponse{}, nil
}

func (f *fakeClient) ContainerExecInspect(_ context.Context, execID string) (types.ContainerExecInspect, error) {
	if f.execInspectFunc != nil {
		return f.execInspectFunc(execID)
	}
	return types.ContainerExecInspect{}, nil
}

func (f *fakeClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	return nil
}
