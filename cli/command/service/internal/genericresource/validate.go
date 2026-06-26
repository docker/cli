package genericresource

import (
	api "github.com/moby/moby/api/types/swarm"
)

// HasResource checks if there is enough "res" in the "resources" argument
func HasResource(res api.GenericResource, resources []api.GenericResource) bool {
	for _, r := range resources {
		if equalResource(r, res) {
			return true
		}
	}
	return false
}

// equalResource matches the resource *type* (named vs discrete), and then kind+value.
func equalResource(a, b api.GenericResource) bool {
	switch {
	case a.NamedResourceSpec != nil && b.NamedResourceSpec != nil:
		return a.NamedResourceSpec.Kind == b.NamedResourceSpec.Kind &&
			a.NamedResourceSpec.Value == b.NamedResourceSpec.Value

	case a.DiscreteResourceSpec != nil && b.DiscreteResourceSpec != nil:
		return a.DiscreteResourceSpec.Kind == b.DiscreteResourceSpec.Kind &&
			a.DiscreteResourceSpec.Value == b.DiscreteResourceSpec.Value
	}
	return false
}
