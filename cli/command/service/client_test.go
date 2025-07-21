package service

import (
	"context"

	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	serviceInspectWithRawFunc func(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error)
	serviceUpdateFunc         func(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, options swarm.ServiceUpdateOptions) (swarm.ServiceUpdateResponse, error)
	serviceListFunc           func(context.Context, swarm.ServiceListOptions) ([]swarm.Service, error)
	taskListFunc              func(context.Context, swarm.TaskListOptions) ([]swarm.Task, error)
	infoFunc                  func(ctx context.Context) (system.Info, error)
	networkInspectFunc        func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error)
	nodeListFunc              func(ctx context.Context, options swarm.NodeListOptions) ([]swarm.Node, error)
}

func (f *fakeClient) NodeList(ctx context.Context, options swarm.NodeListOptions) ([]swarm.Node, error) {
	if f.nodeListFunc != nil {
		return f.nodeListFunc(ctx, options)
	}
	return nil, nil
}

func (f *fakeClient) TaskList(ctx context.Context, options swarm.TaskListOptions) ([]swarm.Task, error) {
	if f.taskListFunc != nil {
		return f.taskListFunc(ctx, options)
	}
	return nil, nil
}

func (f *fakeClient) ServiceInspectWithRaw(ctx context.Context, serviceID string, options swarm.ServiceInspectOptions) (swarm.Service, []byte, error) {
	if f.serviceInspectWithRawFunc != nil {
		return f.serviceInspectWithRawFunc(ctx, serviceID, options)
	}

	return *builders.Service(builders.ServiceID(serviceID)), []byte{}, nil
}

func (f *fakeClient) ServiceList(ctx context.Context, options swarm.ServiceListOptions) ([]swarm.Service, error) {
	if f.serviceListFunc != nil {
		return f.serviceListFunc(ctx, options)
	}

	return nil, nil
}

func (f *fakeClient) ServiceUpdate(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, options swarm.ServiceUpdateOptions) (swarm.ServiceUpdateResponse, error) {
	if f.serviceUpdateFunc != nil {
		return f.serviceUpdateFunc(ctx, serviceID, version, service, options)
	}

	return swarm.ServiceUpdateResponse{}, nil
}

func (f *fakeClient) Info(ctx context.Context) (system.Info, error) {
	if f.infoFunc == nil {
		return system.Info{}, nil
	}
	return f.infoFunc(ctx)
}

func (f *fakeClient) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	if f.networkInspectFunc != nil {
		return f.networkInspectFunc(ctx, networkID, options)
	}
	return network.Inspect{}, nil
}

func newService(id string, name string) swarm.Service {
	return *builders.Service(builders.ServiceID(id), builders.ServiceName(name))
}
