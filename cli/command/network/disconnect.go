package network

import (
	"context"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

type disconnectOptions struct {
	network   string
	container string
	force     bool
}

func newDisconnectCommand(dockerCli command.Cli) *cobra.Command {
	opts := disconnectOptions{}

	cmd := &cobra.Command{
		Use:   "disconnect [OPTIONS] NETWORK CONTAINER",
		Short: "Disconnect a container from a network",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.network = args[0]
			opts.container = args[1]
			return runDisconnect(cmd.Context(), dockerCli.Client(), opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completion.NetworkNames(dockerCli)(cmd, args, toComplete)
			}
			network := args[0]
			return completion.ContainerNames(dockerCli, true, isConnected(network))(cmd, args, toComplete)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the container to disconnect from a network")

	return cmd
}

func runDisconnect(ctx context.Context, apiClient client.NetworkAPIClient, opts disconnectOptions) error {
	return apiClient.NetworkDisconnect(ctx, opts.network, opts.container, opts.force)
}

func isConnected(network string) func(container.Summary) bool {
	return func(ctr container.Summary) bool {
		if ctr.NetworkSettings == nil {
			return false
		}
		_, ok := ctr.NetworkSettings.Networks[network]
		return ok
	}
}

func not(fn func(container.Summary) bool) func(container.Summary) bool {
	return func(ctr container.Summary) bool {
		ok := fn(ctr)
		return !ok
	}
}
