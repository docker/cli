package genericresource

import (
	api "github.com/moby/moby/api/types/swarm"
)

// NewSet creates a set object
func NewSet(key string, vals ...string) []api.GenericResource {
	rs := make([]api.GenericResource, 0, len(vals))
	for _, v := range vals {
		rs = append(rs, NewString(key, v))
	}
	return rs
}

// NewString creates a String resource
func NewString(kind, value string) api.GenericResource {
	return api.GenericResource{
		NamedResourceSpec: &api.NamedGenericResource{
			Kind:  kind,
			Value: value,
		},
	}
}

// NewDiscrete creates a Discrete resource
func NewDiscrete(key string, val int64) api.GenericResource {
	return api.GenericResource{
		DiscreteResourceSpec: &api.DiscreteGenericResource{
			Kind:  key,
			Value: val,
		},
	}
}

// GetResource returns resources from the "resources" parameter matching the kind key
func GetResource(kind string, resources []api.GenericResource) []api.GenericResource {
	var res []api.GenericResource
	for _, r := range resources {
		switch {
		case r.DiscreteResourceSpec != nil && r.DiscreteResourceSpec.Kind == kind:
			res = append(res, r)
		case r.NamedResourceSpec != nil && r.NamedResourceSpec.Kind == kind:
			res = append(res, r)
		}
	}
	return res
}
