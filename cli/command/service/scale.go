package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type scaleOptions struct {
	detach bool
}

func newScaleCommand(dockerCLI command.Cli) *cobra.Command {
	options := &scaleOptions{}

	cmd := &cobra.Command{
		Use:   "scale SERVICE=REPLICAS [SERVICE=REPLICAS...]",
		Short: "Scale one or multiple replicated services",
		Args:  scaleArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScale(cmd.Context(), dockerCLI, options, args)
		},
		ValidArgsFunction:     completeScaleArgs(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	addDetachFlag(flags, &options.detach)
	return cmd
}

func scaleArgs(cmd *cobra.Command, args []string) error {
	if err := cli.RequiresMinArgs(1)(cmd, args); err != nil {
		return err
	}
	for _, arg := range args {
		if k, v, ok := strings.Cut(arg, "="); !ok || k == "" || v == "" {
			return fmt.Errorf(
				"invalid scale specifier '%s'.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
				arg,
				cmd.CommandPath(),
				cmd.UseLine(),
				cmd.Short,
			)
		}
	}
	return nil
}

func runScale(ctx context.Context, dockerCLI command.Cli, options *scaleOptions, args []string) error {
	apiClient := dockerCLI.Client()
	var (
		errs       []error
		serviceIDs = make([]string, 0, len(args))
	)
	for _, arg := range args {
		serviceID, scaleStr, _ := strings.Cut(arg, "=")

		// validate input arg scale number
		scale, err := strconv.ParseUint(scaleStr, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid replicas value %s: %v", serviceID, scaleStr, err))
			continue
		}

		warnings, err := runServiceScale(ctx, apiClient, serviceID, scale)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", serviceID, err))
			continue
		}
		for _, warning := range warnings {
			_, _ = fmt.Fprintln(dockerCLI.Err(), warning)
		}
		_, _ = fmt.Fprintf(dockerCLI.Out(), "%s scaled to %d\n", serviceID, scale)
		serviceIDs = append(serviceIDs, serviceID)
	}

	if len(serviceIDs) > 0 && !options.detach {
		for _, serviceID := range serviceIDs {
			if err := WaitOnService(ctx, dockerCLI, serviceID, false); err != nil {
				errs = append(errs, fmt.Errorf("%s: %v", serviceID, err))
			}
		}
	}
	return errors.Join(errs...)
}

func runServiceScale(ctx context.Context, apiClient client.ServiceAPIClient, serviceID string, scale uint64) (warnings []string, _ error) {
	res, err := apiClient.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}

	serviceMode := &res.Service.Spec.Mode
	switch {
	case serviceMode.Replicated != nil:
		serviceMode.Replicated.Replicas = &scale
	case serviceMode.ReplicatedJob != nil:
		serviceMode.ReplicatedJob.TotalCompletions = &scale
	default:
		return nil, errors.New("scale can only be used with replicated or replicated-job mode")
	}

	response, err := apiClient.ServiceUpdate(ctx, res.Service.ID, client.ServiceUpdateOptions{
		Version: res.Service.Version,
		Spec:    res.Service.Spec,
	})
	if err != nil {
		return nil, err
	}
	return response.Warnings, nil
}

// completeScaleArgs returns a completion function for the args of the scale command.
// It completes service names followed by "=", suppressing the trailing space.
func completeScaleArgs(dockerCli command.Cli) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// reuse the existing logic for configurable completion of service names and IDs.
		completions, directive := completeServiceNames(dockerCli)(cmd, args, toComplete)
		if directive == cobra.ShellCompDirectiveError {
			return completions, directive
		}
		for i, v := range completions {
			completions[i] = v + "="
		}
		return completions, directive | cobra.ShellCompDirectiveNoSpace
	}
}
