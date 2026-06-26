package node

import (
	"context"
	"errors"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/task"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type psOptions struct {
	nodeIDs   []string
	noResolve bool
	noTrunc   bool
	quiet     bool
	format    string
	filter    opts.FilterOpt
}

func newPsCommand(dockerCLI command.Cli) *cobra.Command {
	options := psOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] [NODE...]",
		Short: "List tasks running on one or more nodes, defaults to current node",
		Args:  cli.RequiresMinArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.nodeIDs = []string{"self"}

			if len(args) != 0 {
				options.nodeIDs = args
			}

			return runPs(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     completeNodeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&options.noResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")
	flags.StringVar(&options.format, "format", "", "Pretty-print tasks using a Go template")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display task IDs")

	return cmd
}

func runPs(ctx context.Context, dockerCLI command.Cli, options psOptions) error {
	apiClient := dockerCLI.Client()

	var (
		errs  []error
		tasks = client.TaskListResult{}
	)

	for _, nodeID := range options.nodeIDs {
		nodeRef, err := Reference(ctx, apiClient, nodeID)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		res, err := apiClient.NodeInspect(ctx, nodeRef, client.NodeInspectOptions{})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		filter := options.filter.Value()
		filter.Add("node", res.Node.ID)

		nodeTasks, err := apiClient.TaskList(ctx, client.TaskListOptions{Filters: filter})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		tasks.Items = append(tasks.Items, nodeTasks.Items...)
	}

	format := options.format
	if len(format) == 0 {
		format = task.DefaultFormat(dockerCLI.ConfigFile(), options.quiet)
	}

	if len(errs) == 0 || len(tasks.Items) != 0 {
		if err := task.Print(ctx, dockerCLI, tasks, idresolver.New(apiClient, options.noResolve), !options.noTrunc, options.quiet, format); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
