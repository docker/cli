package swarm

import (
	"context"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

func getStackFilter(namespace string) filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getStackFilterFromOpt(namespace string, opt opts.FilterOpt) filters.Args {
	filter := opt.Value()
	filter.Add("label", convert.LabelNamespace+"="+namespace)
	return filter
}

func getAllStacksFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", convert.LabelNamespace)
	return filter
}

func getStackServices(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Service, error) {
	return apiclient.ServiceList(ctx, types.ServiceListOptions{Filters: getStackFilter(namespace)})
}

func getStackNetworks(ctx context.Context, apiclient client.APIClient, namespace string) ([]network.Summary, error) {
	return apiclient.NetworkList(ctx, network.ListOptions{Filters: getStackFilter(namespace)})
}

func getStackSecrets(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Secret, error) {
	return apiclient.SecretList(ctx, types.SecretListOptions{Filters: getStackFilter(namespace)})
}

func getStackConfigs(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Config, error) {
	return apiclient.ConfigList(ctx, types.ConfigListOptions{Filters: getStackFilter(namespace)})
}

func getStackTasks(ctx context.Context, apiclient client.APIClient, namespace string) ([]swarm.Task, error) {
	return apiclient.TaskList(ctx, types.TaskListOptions{Filters: getStackFilter(namespace)})
}
