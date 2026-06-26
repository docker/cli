package task

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.APIClient
	nodeInspectFunc    func(ref string) (client.NodeInspectResult, error)
	serviceInspectFunc func(ref string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
}

func (cli *fakeClient) NodeInspect(_ context.Context, ref string, _ client.NodeInspectOptions) (client.NodeInspectResult, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc(ref)
	}
	return client.NodeInspectResult{}, nil
}

func (cli *fakeClient) ServiceInspect(_ context.Context, ref string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	if cli.serviceInspectFunc != nil {
		return cli.serviceInspectFunc(ref, options)
	}
	return client.ServiceInspectResult{}, nil
}
