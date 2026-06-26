package stack

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/task"
	flagsHelper "github.com/docker/cli/cli/flags"
	cliopts "github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// psOptions holds docker stack ps options
type psOptions struct {
	filter    cliopts.FilterOpt
	noTrunc   bool
	namespace string
	noResolve bool
	quiet     bool
	format    string
}

func newPsCommand(dockerCLI command.Cli) *cobra.Command {
	opts := psOptions{filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			if err := validateStackName(opts.namespace); err != nil {
				return err
			}
			return runPS(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.BoolVar(&opts.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.noResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.filter, "filter", "f", "Filter output based on conditions provided")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&opts.format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

// runPS is the swarm implementation of docker stack ps
func runPS(ctx context.Context, dockerCLI command.Cli, opts psOptions) error {
	apiClient := dockerCLI.Client()
	res, err := apiClient.TaskList(ctx, client.TaskListOptions{
		Filters: getStackFilterFromOpt(opts.namespace, opts.filter),
	})
	if err != nil {
		return err
	}

	if len(res.Items) == 0 {
		return fmt.Errorf("nothing found in stack: %s", opts.namespace)
	}

	if opts.format == "" {
		opts.format = task.DefaultFormat(dockerCLI.ConfigFile(), opts.quiet)
	}

	return task.Print(ctx, dockerCLI, res, idresolver.New(apiClient, opts.noResolve), !opts.noTrunc, opts.quiet, opts.format)
}
