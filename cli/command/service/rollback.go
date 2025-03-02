package service

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/versions"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newRollbackCommand(dockerCli command.Cli) *cobra.Command {
	options := newServiceOptions()

	cmd := &cobra.Command{
		Use:   "rollback [OPTIONS] SERVICE",
		Short: "Revert changes to a service's configuration",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(cmd.Context(), dockerCli, options, args[0])
		},
		Annotations: map[string]string{"version": "1.31"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompletionFn(dockerCli)(cmd, args, toComplete)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, flagQuiet, "q", false, "Suppress progress output")
	addDetachFlag(flags, &options.detach)

	flags.VisitAll(func(flag *pflag.Flag) {
		// Set a default completion function if none was set. We don't look
		// up if it does already have one set, because Cobra does this for
		// us, and returns an error (which we ignore for this reason).
		_ = cmd.RegisterFlagCompletionFunc(flag.Name, completion.NoComplete)
	})
	return cmd
}

func runRollback(ctx context.Context, dockerCLI command.Cli, options *serviceOptions, serviceID string) error {
	apiClient := dockerCLI.Client()

	service, _, err := apiClient.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	response, err := apiClient.ServiceUpdate(ctx, service.ID, service.Version, service.Spec, types.ServiceUpdateOptions{
		Rollback: "previous", // TODO(thaJeztah): this should have a const defined
	})
	if err != nil {
		return err
	}

	for _, warning := range response.Warnings {
		_, _ = fmt.Fprintln(dockerCLI.Err(), warning)
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), serviceID)

	if options.detach || versions.LessThan(apiClient.ClientVersion(), "1.29") {
		return nil
	}

	return WaitOnService(ctx, dockerCLI, serviceID, options.quiet)
}
