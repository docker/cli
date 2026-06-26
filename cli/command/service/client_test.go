package service

import (
	"context"

	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	serviceInspectFunc func(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	serviceUpdateFunc  func(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
	serviceListFunc    func(context.Context, client.ServiceListOptions) (client.ServiceListResult, error)
	taskListFunc       func(context.Context, client.TaskListOptions) (client.TaskListResult, error)
	infoFunc           func(ctx context.Context) (client.SystemInfoResult, error)
	networkInspectFunc func(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error)
	nodeListFunc       func(ctx context.Context, options client.NodeListOptions) (client.NodeListResult, error)
}

func (f *fakeClient) NodeList(ctx context.Context, options client.NodeListOptions) (client.NodeListResult, error) {
	if f.nodeListFunc != nil {
		return f.nodeListFunc(ctx, options)
	}
	return client.NodeListResult{}, nil
}

func (f *fakeClient) TaskList(ctx context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	if f.taskListFunc != nil {
		return f.taskListFunc(ctx, options)
	}
	return client.TaskListResult{}, nil
}

func (f *fakeClient) ServiceInspect(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if f.serviceInspectFunc != nil {
		return f.serviceInspectFunc(ctx, serviceID, options)
	}

	return client.ServiceInspectResult{
		Service: *builders.Service(builders.ServiceID(serviceID)),
	}, nil
}

func (f *fakeClient) ServiceList(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
	if f.serviceListFunc != nil {
		return f.serviceListFunc(ctx, options)
	}

	return client.ServiceListResult{}, nil
}

func (f *fakeClient) ServiceUpdate(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	if f.serviceUpdateFunc != nil {
		return f.serviceUpdateFunc(ctx, serviceID, options)
	}

	return client.ServiceUpdateResult{}, nil
}

func (f *fakeClient) Info(ctx context.Context, _ client.InfoOptions) (client.SystemInfoResult, error) {
	if f.infoFunc != nil {
		return f.infoFunc(ctx)
	}
	return client.SystemInfoResult{}, nil
}

func (f *fakeClient) NetworkInspect(ctx context.Context, networkID string, options client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	if f.networkInspectFunc != nil {
		return f.networkInspectFunc(ctx, networkID, options)
	}
	return client.NetworkInspectResult{}, nil
}

func newService(id string, name string) swarm.Service {
	return *builders.Service(builders.ServiceID(id), builders.ServiceName(name))
}
