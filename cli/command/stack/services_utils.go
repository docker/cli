package stack

import (
	"context"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// getServices is the swarm implementation of listing stack services
func getServices(ctx context.Context, apiClient client.APIClient, opts serviceListOptions) ([]swarm.Service, error) {
	return apiClient.ServiceList(ctx, client.ServiceListOptions{
		Filters: getStackFilterFromOpt(opts.namespace, opts.filter),
		// When not running "quiet", also get service status (number of running
		// and desired tasks).
		Status: !opts.quiet,
	})
}
