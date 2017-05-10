package json

import (
	"io"

	"github.com/docker/docker/api/types/swarm"
)

func DecodeServiceCreate(body io.Reader) (swarm.ServiceSpec, error) {
	var service swarm.ServiceSpec
	err := decode(body, &service)
	return service, err
}
