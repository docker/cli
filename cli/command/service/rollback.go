package service

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newRollbackCommand(dockerCLI command.Cli) *cobra.Command {
	options := newServiceOptions()

	cmd := &cobra.Command{
		Use:   "rollback [OPTIONS] SERVICE",
		Short: "Revert changes to a service's configuration",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(cmd.Context(), dockerCLI, options, args[0])
		},
		Annotations:           map[string]string{"version": "1.31"},
		ValidArgsFunction:     completeServiceNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, flagQuiet, "q", false, "Suppress progress output")
	addDetachFlag(flags, &options.detach)

	return cmd
}

func runRollback(ctx context.Context, dockerCLI command.Cli, options *serviceOptions, serviceID string) error {
	apiClient := dockerCLI.Client()

	res, err := apiClient.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	response, err := apiClient.ServiceUpdate(ctx, res.Service.ID, client.ServiceUpdateOptions{
		Version:  res.Service.Version,
		Spec:     res.Service.Spec,
		Rollback: "previous", // TODO(thaJeztah): this should have a const defined
	})
	if err != nil {
		return err
	}

	for _, warning := range response.Warnings {
		_, _ = fmt.Fprintln(dockerCLI.Err(), warning)
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), serviceID)

	if options.detach {
		return nil
	}

	return WaitOnService(ctx, dockerCLI, serviceID, options.quiet)
}
