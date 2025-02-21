package stack // import "docker.com/cli/v28/cli/command/stack"

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/service"
	"github.com/docker/cli/v28/cli/command/stack/formatter"
	"github.com/docker/cli/v28/cli/command/stack/options"
	"github.com/docker/cli/v28/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/v28/cli/flags"
	cliopts "github.com/docker/cli/v28/opts"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

func newServicesCommand(dockerCli command.Cli) *cobra.Command {
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
			return RunServices(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.Format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	return cmd
}

// RunServices performs a stack services against the specified swarm cluster
func RunServices(ctx context.Context, dockerCli command.Cli, opts options.Services) error {
	services, err := swarm.GetServices(ctx, dockerCli, opts)
	if err != nil {
		return err
	}
	return formatWrite(dockerCli, services, opts)
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
