package client

import (
	"net/url"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type startconfig struct {
       Config                   *container.Config
       HostConfig       *container.HostConfig
       NetworkingConfig *network.NetworkingConfig
}

// ContainerStart sends a request to the docker daemon to start a container.
func (cli *Client) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	query := url.Values{}
	if len(options.CheckpointID) != 0 {
		query.Set("checkpoint", options.CheckpointID)
	}
	if len(options.CheckpointDir) != 0 {
		query.Set("checkpoint-dir", options.CheckpointDir)
	}

	// add body in request
	ports, portBindings, err := nat.ParsePortSpecs(options.Portmap)

	var hostConfig = &container.HostConfig{
		PortBindings: portBindings,
	}
	var config = &container.Config{
		ExposedPorts: ports,
		Image: "ubuntu",
	}
	var networkingConfig = &network.NetworkingConfig{}
	body := startconfig{
		Config:			  config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
	}
	resp, err := cli.post(ctx, "/containers/"+containerID+"/start", query, body, nil)
//	resp, err := cli.post(ctx, "/containers/"+containerID+"/start", query, nil, nil)
	ensureReaderClosed(resp)
	return err
}
