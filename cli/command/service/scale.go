package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/versions"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type scaleOptions struct {
	detach bool
}

func newScaleCommand(dockerCli command.Cli) *cobra.Command {
	options := &scaleOptions{}

	cmd := &cobra.Command{
		Use:   "scale SERVICE=REPLICAS [SERVICE=REPLICAS...]",
		Short: "Scale one or multiple replicated services",
		Args:  scaleArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScale(cmd.Context(), dockerCli, options, args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompletionFn(dockerCli)(cmd, args, toComplete)
		},
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
			return errors.Errorf(
				"Invalid scale specifier '%s'.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
				arg,
				cmd.CommandPath(),
				cmd.UseLine(),
				cmd.Short,
			)
		}
	}
	return nil
}

func runScale(ctx context.Context, dockerCli command.Cli, options *scaleOptions, args []string) error {
	var errs []string
	var serviceIDs []string

	for _, arg := range args {
		serviceID, scaleStr, _ := strings.Cut(arg, "=")

		// validate input arg scale number
		scale, err := strconv.ParseUint(scaleStr, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid replicas value %s: %v", serviceID, scaleStr, err))
			continue
		}

		if err := runServiceScale(ctx, dockerCli, serviceID, scale); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", serviceID, err))
		} else {
			serviceIDs = append(serviceIDs, serviceID)
		}
	}

	if len(serviceIDs) > 0 {
		if !options.detach && versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.29") {
			for _, serviceID := range serviceIDs {
				if err := WaitOnService(ctx, dockerCli, serviceID, false); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", serviceID, err))
				}
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}

func runServiceScale(ctx context.Context, dockerCli command.Cli, serviceID string, scale uint64) error {
	client := dockerCli.Client()

	service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceMode := &service.Spec.Mode
	switch {
	case serviceMode.Replicated != nil:
		serviceMode.Replicated.Replicas = &scale
	case serviceMode.ReplicatedJob != nil:
		serviceMode.ReplicatedJob.TotalCompletions = &scale
	default:
		return errors.Errorf("scale can only be used with replicated or replicated-job mode")
	}

	response, err := client.ServiceUpdate(ctx, service.ID, service.Version, service.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		return err
	}

	for _, warning := range response.Warnings {
		fmt.Fprintln(dockerCli.Err(), warning)
	}

	fmt.Fprintf(dockerCli.Out(), "%s scaled to %d\n", serviceID, scale)
	return nil
}
