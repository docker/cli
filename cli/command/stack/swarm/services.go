package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types"
)

// RunServices is the swarm implementation of docker stack services
func RunServices(dockerCli command.Cli, opts options.Services) error {
	var (
		err    error
		ctx    = context.Background()
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
		return err
	}

	// if no services in this stack, print message and exit 0
	if len(services) == 0 {
		_, _ = fmt.Fprintf(dockerCli.Err(), "Nothing found in stack: %s\n", opts.Namespace)
		return nil
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
			return err
		}
	}

	format := opts.Format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ServicesFormat) > 0 && !opts.Quiet {
			format = dockerCli.ConfigFile().ServicesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: service.NewListFormat(format, opts.Quiet),
	}
	return service.ListFormatWrite(servicesCtx, services)
}
