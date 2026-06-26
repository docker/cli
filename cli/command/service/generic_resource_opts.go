package service

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command/service/internal/genericresource"
	"github.com/moby/moby/api/types/swarm"
)

// GenericResource is a concept that a user can use to advertise user-defined
// resources on a node and thus better place services based on these resources.
// E.g: NVIDIA GPUs, Intel FPGAs, ...
// See https://github.com/moby/swarmkit/blob/de950a7ed842c7b7e47e9451cde9bf8f96031894/design/generic_resources.md

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

	swarmResources, err := genericresource.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("invalid generic resource specification: %w", err)
	}

	for _, res := range swarmResources {
		if res.NamedResourceSpec != nil {
			return nil, fmt.Errorf("invalid generic-resource request `%s=%s`, Named Generic Resources is not supported for service create or update",
				res.NamedResourceSpec.Kind, res.NamedResourceSpec.Value,
			)
		}
	}

	return swarmResources, nil
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
	l := make([]swarm.GenericResource, 0, len(genericRes))

	for _, res := range genericRes {
		l = append(l, res)
	}

	return l
}
