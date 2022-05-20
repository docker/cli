package network

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
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
			return runDisconnect(dockerCli, opts)
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

func runDisconnect(dockerCli command.Cli, opts disconnectOptions) error {
	client := dockerCli.Client()

	return client.NetworkDisconnect(context.Background(), opts.network, opts.container, opts.force)
}

func isConnected(network string) func(types.Container) bool {
	return func(container types.Container) bool {
		if container.NetworkSettings == nil {
			return false
		}
		_, ok := container.NetworkSettings.Networks[network]
		return ok
	}
}

func not(fn func(types.Container) bool) func(types.Container) bool {
	return func(container types.Container) bool {
		ok := fn(container)
		return !ok
	}
}
