package service

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/versions"
	"github.com/spf13/cobra"
)

func newRollbackCommand(dockerCli command.Cli) *cobra.Command {
	options := newServiceOptions()

	cmd := &cobra.Command{
		Use:   "rollback [OPTIONS] SERVICE",
		Short: "Revert changes to a service's configuration",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(dockerCli, options, args[0])
		},
		Annotations: map[string]string{"version": "1.31"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, flagQuiet, "q", false, "Suppress progress output")
	addDetachFlag(flags, &options.detach)

	return cmd
}

func runRollback(dockerCli command.Cli, options *serviceOptions, serviceID string) error {
	apiClient := dockerCli.Client()
	ctx := context.Background()

	service, _, err := apiClient.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	spec := &service.Spec
	updateOpts := types.ServiceUpdateOptions{
		Rollback: "previous",
	}

	response, err := apiClient.ServiceUpdate(ctx, service.ID, service.Version, *spec, updateOpts)
	if err != nil {
		return err
	}

	for _, warning := range response.Warnings {
		fmt.Fprintln(dockerCli.Err(), warning)
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", serviceID)

	if options.detach || versions.LessThan(apiClient.ClientVersion(), "1.29") {
		return nil
	}

	return waitOnService(ctx, dockerCli, serviceID, options.quiet)
}
