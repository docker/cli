package node

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	infoFunc           func() (client.SystemInfoResult, error)
	nodeInspectFunc    func() (client.NodeInspectResult, error)
	nodeListFunc       func() (client.NodeListResult, error)
	nodeRemoveFunc     func() (client.NodeRemoveResult, error)
	nodeUpdateFunc     func(nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error)
	taskInspectFunc    func(taskID string) (client.TaskInspectResult, error)
	taskListFunc       func(options client.TaskListOptions) (client.TaskListResult, error)
	serviceInspectFunc func(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error)
}

func (cli *fakeClient) NodeInspect(context.Context, string, client.NodeInspectOptions) (client.NodeInspectResult, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc()
	}
	return client.NodeInspectResult{}, nil
}

func (cli *fakeClient) NodeList(context.Context, client.NodeListOptions) (client.NodeListResult, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc()
	}
	return client.NodeListResult{}, nil
}

func (cli *fakeClient) NodeRemove(context.Context, string, client.NodeRemoveOptions) (client.NodeRemoveResult, error) {
	if cli.nodeRemoveFunc != nil {
		return cli.nodeRemoveFunc()
	}
	return client.NodeRemoveResult{}, nil
}

func (cli *fakeClient) NodeUpdate(_ context.Context, nodeID string, options client.NodeUpdateOptions) (client.NodeUpdateResult, error) {
	if cli.nodeUpdateFunc != nil {
		return cli.nodeUpdateFunc(nodeID, options)
	}
	return client.NodeUpdateResult{}, nil
}

func (cli *fakeClient) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc()
	}
	return client.SystemInfoResult{}, nil
}

func (cli *fakeClient) TaskInspect(_ context.Context, taskID string, _ client.TaskInspectOptions) (client.TaskInspectResult, error) {
	if cli.taskInspectFunc != nil {
		return cli.taskInspectFunc(taskID)
	}
	return client.TaskInspectResult{}, nil
}

func (cli *fakeClient) TaskList(_ context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	if cli.taskListFunc != nil {
		return cli.taskListFunc(options)
	}
	return client.TaskListResult{}, nil
}

func (cli *fakeClient) ServiceInspect(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if cli.serviceInspectFunc != nil {
		return cli.serviceInspectFunc(ctx, serviceID, opts)
	}
	return client.ServiceInspectResult{}, nil
}
