package task

import (
	"context"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.APIClient
	nodeInspectWithRaw    func(ref string) (swarm.Node, []byte, error)
	serviceInspectWithRaw func(ref string, options client.ServiceInspectOptions) (swarm.Service, []byte, error)
}

func (cli *fakeClient) NodeInspectWithRaw(_ context.Context, ref string) (swarm.Node, []byte, error) {
	if cli.nodeInspectWithRaw != nil {
		return cli.nodeInspectWithRaw(ref)
	}
	return swarm.Node{}, nil, nil
}

func (cli *fakeClient) ServiceInspectWithRaw(_ context.Context, ref string, options client.ServiceInspectOptions) (swarm.Service, []byte, error) {
	if cli.serviceInspectWithRaw != nil {
		return cli.serviceInspectWithRaw(ref, options)
	}
	return swarm.Service{}, nil, nil
}
