package service

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/swarm"
	swarmapi "github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
)

// GenericResource is a concept that a user can use to advertise user-defined
// resources on a node and thus better place services based on these resources.
// E.g: NVIDIA GPUs, Intel FPGAs, ...
// See https://github.com/docker/swarmkit/blob/master/design/generic_resources.md

// ValidateSingleGenericResource validates that a single entry in the
// generic resource list is valid.
// i.e 'GPU=UID1' is valid however 'GPU:UID1' or 'UID1' isn't
func ValidateSingleGenericResource(val string) (string, error) {
	if strings.Count(val, "=") < 1 {
		return "", fmt.Errorf("invalid generic-resource format `%s` expected `name=value`", val)
	}

	return val, nil
}

// ParseGenericResources parses an array of Generic resourceResources
// Requesting Named Generic Resources for a service is not supported this
// is filtered here.
func ParseGenericResources(value []string) ([]swarm.GenericResource, error) {
	if len(value) == 0 {
		return nil, nil
	}

	resources, err := genericresource.Parse(value)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid generic resource specification")
	}

	swarmResources := genericResourcesFromGRPC(resources)
	for _, res := range swarmResources {
		if res.NamedResourceSpec != nil {
			return nil, fmt.Errorf("invalid generic-resource request `%s=%s`, Named Generic Resources is not supported for service create or update", res.NamedResourceSpec.Kind, res.NamedResourceSpec.Value)
		}
	}

	return swarmResources, nil
}

// genericResourcesFromGRPC converts a GRPC GenericResource to a GenericResource
func genericResourcesFromGRPC(genericRes []*swarmapi.GenericResource) []swarm.GenericResource {
	var generic []swarm.GenericResource
	for _, res := range genericRes {
		var current swarm.GenericResource

		switch r := res.Resource.(type) {
		case *swarmapi.GenericResource_DiscreteResourceSpec:
			current.DiscreteResourceSpec = &swarm.DiscreteGenericResource{
				Kind:  r.DiscreteResourceSpec.Kind,
				Value: r.DiscreteResourceSpec.Value,
			}
		case *swarmapi.GenericResource_NamedResourceSpec:
			current.NamedResourceSpec = &swarm.NamedGenericResource{
				Kind:  r.NamedResourceSpec.Kind,
				Value: r.NamedResourceSpec.Value,
			}
		}

		generic = append(generic, current)
	}

	return generic
}

func buildGenericResourceMap(genericRes []swarm.GenericResource) (map[string]swarm.GenericResource, error) {
	m := make(map[string]swarm.GenericResource)

	for _, res := range genericRes {
		if res.DiscreteResourceSpec == nil {
			return nil, fmt.Errorf("invalid generic-resource `%+v` for service task", res)
		}

		_, ok := m[res.DiscreteResourceSpec.Kind]
		if ok {
			return nil, fmt.Errorf("duplicate generic-resource `%+v` for service task", res.DiscreteResourceSpec.Kind)
		}

		m[res.DiscreteResourceSpec.Kind] = res
	}

	return m, nil
}

func buildGenericResourceList(genericRes map[string]swarm.GenericResource) []swarm.GenericResource {
	var l []swarm.GenericResource

	for _, res := range genericRes {
		l = append(l, res)
	}

	return l
}
