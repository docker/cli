package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// RunServices is the swarm implementation of docker stack services
func RunServices(dockerCli command.Cli, opts options.Services) error {
	ctx := context.Background()
	client := dockerCli.Client()

	filter := getStackFilterFromOpt(opts.Namespace, opts.Filter)
	services, err := client.ServiceList(ctx, types.ServiceListOptions{Filters: filter})
	if err != nil {
		return err
	}

	// if no services in this stack, print message and exit 0
	if len(services) == 0 {
		fmt.Fprintf(dockerCli.Err(), "Nothing found in stack: %s\n", opts.Namespace)
		return nil
	}

	info := map[string]service.ListInfo{}
	if !opts.Quiet {
		taskFilter := filters.NewArgs()
		for _, service := range services {
			taskFilter.Add("service", service.ID)
		}

		tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
		if err != nil {
			return err
		}

		nodes, err := client.NodeList(ctx, types.NodeListOptions{})
		if err != nil {
			return err
		}

		info = service.GetServicesStatus(services, nodes, tasks)
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
	return service.ListFormatWrite(servicesCtx, services, info)
}
