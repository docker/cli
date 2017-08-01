package builders

import (
	"github.com/docker/docker/api/types"
)

// NetworkResource creates a network resource with default values.
// Any number of networkResource function builder can be pass to modify the existing value.
// feel free to add another builder func if you need to override another value
func NetworkResource(builders ...func(resource *types.NetworkResource)) *types.NetworkResource {
	resource := &types.NetworkResource{}

	for _, builder := range builders {
		builder(resource)
	}
	return resource
}

// NetworkResourceName sets the name of the resource network
func NetworkResourceName(name string) func(networkResource *types.NetworkResource) {
	return func(networkResource *types.NetworkResource) {
		networkResource.Name = name
	}
}

// NetworkResourceID sets the ID of the resource network
func NetworkResourceID(id string) func(networkResource *types.NetworkResource) {
	return func(networkResource *types.NetworkResource) {
		networkResource.ID = id
	}
}

// NetworkResourceDriver sets the driver of the resource network
func NetworkResourceDriver(name string) func(networkResource *types.NetworkResource) {
	return func(networkResource *types.NetworkResource) {
		networkResource.Driver = name
	}
}

// NetworkResourceScope sets the Scope of the resource network
func NetworkResourceScope(scope string) func(networkResource *types.NetworkResource) {
	return func(networkResource *types.NetworkResource) {
		networkResource.Scope = scope
	}
}
