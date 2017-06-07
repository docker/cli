package container

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	containerInspectFunc func(string) (types.ContainerJSON, error)
}

func (cli *fakeClient) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	if cli.containerInspectFunc != nil {
		return cli.containerInspectFunc(containerID)
	}
	return types.ContainerJSON{}, nil
}
