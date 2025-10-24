package idresolver

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	nodeInspectFunc    func(string) (client.NodeInspectResult, error)
	serviceInspectFunc func(string) (client.ServiceInspectResult, error)
}

func (cli *fakeClient) NodeInspect(_ context.Context, nodeID string, _ client.NodeInspectOptions) (client.NodeInspectResult, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc(nodeID)
	}
	return client.NodeInspectResult{}, nil
}

func (cli *fakeClient) ServiceInspect(_ context.Context, serviceID string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if cli.serviceInspectFunc != nil {
		return cli.serviceInspectFunc(serviceID)
	}
	return client.ServiceInspectResult{}, nil
}
