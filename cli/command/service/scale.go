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
			return runScale(dockerCli, options, args)
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
		if parts := strings.SplitN(arg, "=", 2); len(parts) != 2 {
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

func runScale(dockerCli command.Cli, options *scaleOptions, args []string) error {
	var errs []string
	var serviceIDs []string
	ctx := context.Background()

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		serviceID, scaleStr := parts[0], parts[1]

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
				if err := waitOnService(ctx, dockerCli, serviceID, false); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", serviceID, err))
				}
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Errorf(strings.Join(errs, "\n"))
}

func runServiceScale(ctx context.Context, dockerCli command.Cli, serviceID string, scale uint64) error {
	client := dockerCli.Client()

	service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceMode := &service.Spec.Mode
	if serviceMode.Replicated == nil {
		return errors.Errorf("scale can only be used with replicated mode")
	}

	serviceMode.Replicated.Replicas = &scale

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
