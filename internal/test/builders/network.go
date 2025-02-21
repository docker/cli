package builders // import "docker.com/cli/v28/internal/test/builders"

import (
	"github.com/docker/docker/api/types/network"
)

// NetworkResource creates a network resource with default values.
// Any number of networkResource function builder can be pass to modify the existing value.
// feel free to add another builder func if you need to override another value
func NetworkResource(builders ...func(resource *network.Summary)) *network.Summary {
	resource := &network.Summary{}

	for _, builder := range builders {
		builder(resource)
	}
	return resource
}

// NetworkResourceName sets the name of the resource network
func NetworkResourceName(name string) func(networkResource *network.Summary) {
	return func(networkResource *network.Summary) {
		networkResource.Name = name
	}
}

// NetworkResourceID sets the ID of the resource network
func NetworkResourceID(id string) func(networkResource *network.Summary) {
	return func(networkResource *network.Summary) {
		networkResource.ID = id
	}
}

// NetworkResourceDriver sets the driver of the resource network
func NetworkResourceDriver(name string) func(networkResource *network.Summary) {
	return func(networkResource *network.Summary) {
		networkResource.Driver = name
	}
}

// NetworkResourceScope sets the Scope of the resource network
func NetworkResourceScope(scope string) func(networkResource *network.Summary) {
	return func(networkResource *network.Summary) {
		networkResource.Scope = scope
	}
}
