package stack

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/legacy/kubernetes"
	"github.com/docker/cli/cli/command/stack/legacy/swarm"
	"github.com/docker/cli/cli/command/stack/options"
	cliopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/filters"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/stacks/pkg/compose/convert"
	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
	"github.com/pkg/errors"
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
	flags.StringVar(&opts.Format, "format", "", "Pretty-print services using a Go template")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunServices performs a stack services against the specified orchestrator
func RunServices(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Services) error {
	if hasServerSideStacks(dockerCli) {
		return RunServerSideServices(dockerCli, flags, commonOrchestrator, opts)
	}
	return runLegacyOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunServices(dockerCli, opts) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunServices(kli, opts) })
}

func RunServerSideServices(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Services) error {
	ctx := context.Background()

	stack, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), opts.Namespace)
	if err != nil {
		return err
	}

	// if no services in this stack, print message and exit 0
	if len(stack.StackResources.Services) == 0 {
		fmt.Fprintf(dockerCli.Err(), "no services found in stack: %s\n", opts.Namespace)
		return nil
	}

	services, info, err := convertToServices(stack, opts.Filter.Value())
	if err != nil {
		return err
	}
	if opts.Quiet {
		info = map[string]service.ListInfo{}
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

// convertToServices will take a stack and map it to the swarm types so we
// can leverage the service formatter logic
func convertToServices(stack types.Stack, filter filters.Args) ([]swarmtypes.Service, map[string]service.ListInfo, error) {
	result := []swarmtypes.Service{}
	infos := make(map[string]service.ListInfo, len(stack.Spec.Services))

	for _, spec := range stack.Spec.Services {
		// Harden for potentially malformed server-side responses
		if _, ok := stack.StackResources.Services[spec.Name]; !ok {
			return nil, nil, fmt.Errorf("unable to find stack resource for service %s", spec.Name)
		}
		if _, ok := stack.Status.ServicesStatus[spec.Name]; !ok {
			return nil, nil, fmt.Errorf("unable to find stack status for service %s", spec.Name)
		}

		if skipByFilters(spec, stack, filter) {
			continue
		}

		// Note: the convert.Service routine is more heavyweight than
		// we likely need (and pulls in quite a bit of vendor
		// dependencies) but makes this conversion simpler to code.  We
		// might consider a refinement to only convert the portions
		// needed for rendering the standard fields in the swarm
		// service formatter
		namespace := convert.NewNamespace(spec.Name)
		serviceSpec, err := convert.Service(
			namespace,
			spec,
			stack.Spec.Networks,
			stack.Spec.Volumes,
			nil, // Omitting Secrets - type mismatch
			nil, // Omitting Configs - type mismatch
		)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to convert service %s", spec.Name)
		}
		svc := swarmtypes.Service{
			ID:   stack.StackResources.Services[spec.Name].ID,
			Spec: serviceSpec,
		}
		infos[svc.ID] = service.ListInfo{
			Mode: spec.Deploy.Mode,
			Replicas: fmt.Sprintf("%d/%d",
				stack.Status.ServicesStatus[spec.Name].RunningTasks,
				stack.Status.ServicesStatus[spec.Name].DesiredTasks),
		}
		result = append(result, svc)
	}
	return result, infos, nil
}

func skipByFilters(serviceSpec composetypes.ServiceConfig, stack types.Stack, filter filters.Args) bool {
	// No filters, include everything
	if filter.Len() == 0 {
		return false
	}
	// Include if any filters match
	if filter.Contains("id") {
		if filter.FuzzyMatch("id", stack.StackResources.Services[serviceSpec.Name].ID) {
			return false
		}
	}
	if filter.Contains("mode") {
		if filter.ExactMatch("mode", serviceSpec.Deploy.Mode) {
			return false
		}
	}
	if filter.Contains("name") {
		if filter.FuzzyMatch("name", serviceSpec.Name) {
			return false
		}
	}
	if filter.Contains("label") {
		for key, value := range serviceSpec.Labels {
			if filter.FuzzyMatch("label", key+"="+value) {
				return false
			}

		}
	}
	// No filters matched, skip the service
	return true
}
