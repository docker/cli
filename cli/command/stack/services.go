package stack

import (
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/cli/flags"
	cliopts "github.com/docker/cli/opts"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newServicesCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
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
			return RunServices(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.Format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunServices performs a stack services against the specified orchestrator
func RunServices(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Services) error {
	services, err := GetServices(dockerCli, flags, commonOrchestrator, opts)
	if err != nil {
		return err
	}
	return formatWrite(dockerCli, services, opts)
}

// GetServices returns the services for the specified orchestrator
func GetServices(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Services) ([]swarmtypes.Service, error) {
	switch {
	case commonOrchestrator.HasAll():
		return nil, errUnsupportedAllOrchestrator
	case commonOrchestrator.HasKubernetes():
		kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(flags, commonOrchestrator))
		if err != nil {
			return nil, err
		}
		return kubernetes.GetServices(kli, opts)
	default:
		return swarm.GetServices(dockerCli, opts)
	}
}

func formatWrite(dockerCli command.Cli, services []swarmtypes.Service, opts options.Services) error {
	// if no services in the stack, print message and exit 0
	if len(services) == 0 {
		_, _ = fmt.Fprintf(dockerCli.Err(), "Nothing found in stack: %s\n", opts.Namespace)
		return nil
	}
	sort.Slice(services, func(i, j int) bool {
		return sortorder.NaturalLess(services[i].Spec.Name, services[j].Spec.Name)
	})

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
