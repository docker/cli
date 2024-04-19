package service

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List services",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	var (
		apiClient = dockerCLI.Client()
		err       error
	)

	listOpts := types.ServiceListOptions{
		Filters: options.filter.Value(),
		// When not running "quiet", also get service status (number of running
		// and desired tasks). Note that this is only supported on API v1.41 and
		// up; older API versions ignore this option, and we will have to collect
		// the information manually below.
		Status: !options.quiet,
	}

	services, err := apiClient.ServiceList(ctx, listOpts)
	if err != nil {
		return err
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
		services, err = AppendServiceStatus(ctx, apiClient, services)
		if err != nil {
			return err
		}
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ServicesFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().ServicesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: NewListFormat(format, options.quiet),
	}
	return ListFormatWrite(servicesCtx, services)
}

// AppendServiceStatus propagates the ServiceStatus field for "services".
//
// If API version v1.41 or up is used, this information is already set by the
// daemon. On older API versions, we need to do some extra requests to get
// that information. Theoretically, this function can be skipped based on API
// version, however, some of our unit tests don't set the API version, and
// there may be other situations where the client uses the "default" version.
// To take these situations into account, we do a quick check for services
// that don't have ServiceStatus set, and perform a lookup for those.
func AppendServiceStatus(ctx context.Context, c client.APIClient, services []swarm.Service) ([]swarm.Service, error) {
	status := map[string]*swarm.ServiceStatus{}
	taskFilter := filters.NewArgs()
	for i, s := range services {
		// there is no need in this switch to check for job modes. jobs are not
		// supported until after ServiceStatus was introduced.
		switch {
		case s.ServiceStatus != nil:
			// Server already returned service-status, so we don't
			// have to look-up tasks for this service.
			continue
		case s.Spec.Mode.Replicated != nil:
			// For replicated services, set the desired number of tasks;
			// that way we can present this information in case we're unable
			// to get a list of tasks from the server.
			services[i].ServiceStatus = &swarm.ServiceStatus{DesiredTasks: *s.Spec.Mode.Replicated.Replicas}
			status[s.ID] = &swarm.ServiceStatus{}
			taskFilter.Add("service", s.ID)
		case s.Spec.Mode.Global != nil:
			// No such thing as number of desired tasks for global services
			services[i].ServiceStatus = &swarm.ServiceStatus{}
			status[s.ID] = &swarm.ServiceStatus{}
			taskFilter.Add("service", s.ID)
		default:
			// Unknown task type
		}
	}
	if len(status) == 0 {
		// All services have their ServiceStatus set, so we're done
		return services, nil
	}

	tasks, err := c.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return services, nil
	}
	activeNodes, err := getActiveNodes(ctx, c)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if status[task.ServiceID] == nil {
			// This should not happen in practice; either all services have
			// a ServiceStatus set, or none of them.
			continue
		}
		// TODO: this should only be needed for "global" services. Replicated
		// services have `Spec.Mode.Replicated.Replicas`, which should give this value.
		if task.DesiredState != swarm.TaskStateShutdown {
			status[task.ServiceID].DesiredTasks++
		}
		if _, nodeActive := activeNodes[task.NodeID]; nodeActive && task.Status.State == swarm.TaskStateRunning {
			status[task.ServiceID].RunningTasks++
		}
	}

	for i, service := range services {
		if s := status[service.ID]; s != nil {
			services[i].ServiceStatus = s
		}
	}
	return services, nil
}

func getActiveNodes(ctx context.Context, c client.NodeAPIClient) (map[string]struct{}, error) {
	nodes, err := c.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}
