package stack

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/service"
	flagsHelper "github.com/docker/cli/cli/flags"
	cliopts "github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// serviceListOptions holds docker stack services options
type serviceListOptions = struct {
	quiet     bool
	format    string
	filter    cliopts.FilterOpt
	namespace string
}

func newServicesCommand(dockerCLI command.Cli) *cobra.Command {
	opts := serviceListOptions{filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "services [OPTIONS] STACK",
		Short: "List the services in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespace = args[0]
			if err := validateStackName(opts.namespace); err != nil {
				return err
			}
			return runServices(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&opts.filter, "filter", "f", "Filter output based on conditions provided")
	return cmd
}

// runServices performs a stack services against the specified swarm cluster
func runServices(ctx context.Context, dockerCLI command.Cli, opts serviceListOptions) error {
	res, err := dockerCLI.Client().ServiceList(ctx, client.ServiceListOptions{
		Filters: getStackFilterFromOpt(opts.namespace, opts.filter),
		// When not running "quiet", also get service status (number of running
		// and desired tasks).
		Status: !opts.quiet,
	})
	if err != nil {
		return err
	}
	return formatWrite(dockerCLI, res, opts)
}

func formatWrite(dockerCLI command.Cli, services client.ServiceListResult, opts serviceListOptions) error {
	// if no services in the stack, print message and exit 0
	if len(services.Items) == 0 {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "Nothing found in stack:", opts.namespace)
		return nil
	}
	sort.Slice(services.Items, func(i, j int) bool {
		return sortorder.NaturalLess(services.Items[i].Spec.Name, services.Items[j].Spec.Name)
	})

	f := opts.format
	if len(f) == 0 {
		if len(dockerCLI.ConfigFile().ServicesFormat) > 0 && !opts.quiet {
			f = dockerCLI.ConfigFile().ServicesFormat
		} else {
			f = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: service.NewListFormat(f, opts.quiet),
	}
	return service.ListFormatWrite(servicesCtx, services)
}
