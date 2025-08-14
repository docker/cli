package stack

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/service"
	"github.com/docker/cli/cli/command/internal/stack/formatter"
	"github.com/docker/cli/cli/command/internal/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/cli/flags"
	cliopts "github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	swarmtypes "github.com/moby/moby/api/types/swarm"
	"github.com/spf13/cobra"
)

func newServicesCommand(dockerCLI cli.Cli) *cobra.Command {
	opts := options.Services{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "services [OPTIONS] STACK",
		Short: "List the services in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}
			return RunServices(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCLI)(cmd, args, toComplete)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.Format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	return cmd
}

// RunServices performs a stack services against the specified swarm cluster
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func RunServices(ctx context.Context, dockerCLI cli.Cli, opts options.Services) error {
	return runServices(ctx, dockerCLI, opts)
}

func runServices(ctx context.Context, dockerCLI cli.Cli, opts options.Services) error {
	services, err := swarm.GetServices(ctx, dockerCLI, opts)
	if err != nil {
		return err
	}
	return formatWrite(dockerCLI, services, opts)
}

func formatWrite(dockerCLI command.Cli, services []swarmtypes.Service, opts options.Services) error {
	// if no services in the stack, print message and exit 0
	if len(services) == 0 {
		_, _ = fmt.Fprintln(dockerCLI.Err(), "Nothing found in stack:", opts.Namespace)
		return nil
	}
	sort.Slice(services, func(i, j int) bool {
		return sortorder.NaturalLess(services[i].Spec.Name, services[j].Spec.Name)
	})

	f := opts.Format
	if len(f) == 0 {
		if len(dockerCLI.ConfigFile().ServicesFormat) > 0 && !opts.Quiet {
			f = dockerCLI.ConfigFile().ServicesFormat
		} else {
			f = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: service.NewListFormat(f, opts.Quiet),
	}
	return service.ListFormatWrite(servicesCtx, services)
}
