package swarm

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

// GetServices is the swarm implementation of listing stack services
func GetServices(ctx context.Context, dockerCli command.Cli, opts options.Services) ([]swarm.Service, error) {
	var (
		err    error
		client = dockerCli.Client()
	)

	listOpts := types.ServiceListOptions{
		Filters: getStackFilterFromOpt(opts.Namespace, opts.Filter),
		// When not running "quiet", also get service status (number of running
		// and desired tasks). Note that this is only supported on API v1.41 and
		// up; older API versions ignore this option, and we will have to collect
		// the information manually below.
		Status: !opts.Quiet,
	}

	services, err := client.ServiceList(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if listOpts.Status {
		// Now that a request was made, we know what API version was used (either
		// through configuration, or after client and daemon negotiated a version).
		// If API version v1.41 or up was used; the daemon should already have done
		// the legwork for us, and we don't have to calculate the number of desired
		// and running tasks. On older API versions, we need to do some extra requests
		// to get that information.
		//
		// So theoretically, this step can be skipped based on API version, however,
		// some of our unit tests don't set the API version, and there may be other
		// situations where the client uses the "default" version. To account for
		// these situations, we do a quick check for services that do not have
		// a ServiceStatus set, and perform a lookup for those.
		services, err = service.AppendServiceStatus(ctx, client, services)
		if err != nil {
			return nil, err
		}
	}
	return services, nil
}
