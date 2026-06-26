package network

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type disconnectOptions struct {
	force bool
}

func newDisconnectCommand(dockerCLI command.Cli) *cobra.Command {
	opts := disconnectOptions{}

	cmd := &cobra.Command{
		Use:   "disconnect [OPTIONS] NETWORK CONTAINER",
		Short: "Disconnect a container from a network",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			network := args[0]
			_, err := dockerCLI.Client().NetworkDisconnect(cmd.Context(), network, client.NetworkDisconnectOptions{
				Container: args[1],
				Force:     opts.force,
			})
			return err
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completion.NetworkNames(dockerCLI)(cmd, args, toComplete)
			}
			nw := args[0]
			return completion.ContainerNames(dockerCLI, true, isConnected(nw))(cmd, args, toComplete)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the container to disconnect from a network")

	return cmd
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
