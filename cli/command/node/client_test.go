package node

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	infoFunc           func() (system.Info, error)
	nodeInspectFunc    func() (swarm.Node, []byte, error)
	nodeListFunc       func() ([]swarm.Node, error)
	nodeRemoveFunc     func() error
	nodeUpdateFunc     func(nodeID string, version swarm.Version, node swarm.NodeSpec) error
	taskInspectFunc    func(taskID string) (swarm.Task, []byte, error)
	taskListFunc       func(options types.TaskListOptions) ([]swarm.Task, error)
	serviceInspectFunc func(ctx context.Context, serviceID string, opts types.ServiceInspectOptions) (swarm.Service, []byte, error)
}

func (cli *fakeClient) NodeInspectWithRaw(context.Context, string) (swarm.Node, []byte, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc()
	}
	return swarm.Node{}, []byte{}, nil
}

func (cli *fakeClient) NodeList(context.Context, types.NodeListOptions) ([]swarm.Node, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc()
	}
	return []swarm.Node{}, nil
}

func (cli *fakeClient) NodeRemove(context.Context, string, types.NodeRemoveOptions) error {
	if cli.nodeRemoveFunc != nil {
		return cli.nodeRemoveFunc()
	}
	return nil
}

func (cli *fakeClient) NodeUpdate(_ context.Context, nodeID string, version swarm.Version, node swarm.NodeSpec) error {
	if cli.nodeUpdateFunc != nil {
		return cli.nodeUpdateFunc(nodeID, version, node)
	}
	return nil
}

func (cli *fakeClient) Info(context.Context) (system.Info, error) {
	if cli.infoFunc != nil {
		return cli.infoFunc()
	}
	return system.Info{}, nil
}

func (cli *fakeClient) TaskInspectWithRaw(_ context.Context, taskID string) (swarm.Task, []byte, error) {
	if cli.taskInspectFunc != nil {
		return cli.taskInspectFunc(taskID)
	}
	return swarm.Task{}, []byte{}, nil
}

func (cli *fakeClient) TaskList(_ context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	if cli.taskListFunc != nil {
		return cli.taskListFunc(options)
	}
	return []swarm.Task{}, nil
}

func (cli *fakeClient) ServiceInspectWithRaw(ctx context.Context, serviceID string, opts types.ServiceInspectOptions) (swarm.Service, []byte, error) {
	if cli.serviceInspectFunc != nil {
		return cli.serviceInspectFunc(ctx, serviceID, opts)
	}
	return swarm.Service{}, []byte{}, nil
}
