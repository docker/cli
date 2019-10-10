package swarm

import (
	"context"
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"vbom.ml/util/sortorder"
)

// GetServices is the swarm implementation of listing stack services
func GetServices(dockerCli command.Cli, opts options.Services) ([]swarm.Service, map[string]service.ListInfo, error) {
	ctx := context.Background()
	client := dockerCli.Client()

	filter := getStackFilterFromOpt(opts.Namespace, opts.Filter)
	services, err := client.ServiceList(ctx, types.ServiceListOptions{Filters: filter})
	if err != nil {
		return nil, nil, err
	}
	if len(services) == 0 {
		return []swarm.Service{}, nil, nil
	}

	sort.Slice(services, func(i, j int) bool {
		return sortorder.NaturalLess(services[i].Spec.Name, services[j].Spec.Name)
	})

	info := map[string]service.ListInfo{}
	if !opts.Quiet {
		taskFilter := filters.NewArgs()
		for _, service := range services {
			taskFilter.Add("service", service.ID)
		}

		tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
		if err != nil {
			return nil, nil, err
		}

		nodes, err := client.NodeList(ctx, types.NodeListOptions{})
		if err != nil {
			return nil, nil, err
		}

		info = service.GetServicesStatus(services, nodes, tasks)
	}
	return services, info, nil
}
