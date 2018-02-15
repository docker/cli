package json

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/runconfig"
)

type ContainerCreateBody struct {
	*container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

// DecodeContainers decodes a json request body and returns a list of container
// summaries
func DecodeContainers(body io.Reader) ([]types.Container, error) {
	var cs []types.Container
	return cs, decode(body, &cs)
}

// DecodeContainer decodes a json request body and returns a container object
func DecodeContainerCreate(body io.Reader) (ContainerCreateBody, error) {
	var container ContainerCreateBody
	return container, decode(body, &container)
}

// DecodeContainer decodes a json request body and returns a container object
func DecodeContainer(body io.Reader) (types.ContainerJSON, error) {
	var container types.ContainerJSON
	return container, decode(body, &container)
}

// DecodeContainerStart decodes a json request body and returns a
// ContainerConfigWrapper. The body may be either an a legacy body with a JSON config,
// or docker-compose passing {} instead of the empty body.
func DecodeContainerStart(body io.Reader) (*runconfig.ContainerConfigWrapper, error) {
	var wrapper runconfig.ContainerConfigWrapper
	err := decode(body, &wrapper)
	switch err {
	case io.EOF:
		return nil, nil
	case nil:
		return &wrapper, nil
	default:
		return nil, err
	}
}
