package network

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm NETWORK [NETWORK...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more networks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCli, args, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Do not error if the network does not exist")
	return cmd
}

const ingressWarning = "WARNING! Before removing the routing-mesh network, " +
	"make sure all the nodes in your swarm run the same docker engine version. " +
	"Otherwise, removal may not be effective and functionality of newly create " +
	"ingress networks will be impaired.\nAre you sure you want to continue?"

func runRemove(dockerCli command.Cli, networks []string, opts *removeOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()
	status := 0

	for _, name := range networks {
		if nw, _, err := client.NetworkInspectWithRaw(ctx, name, types.NetworkInspectOptions{}); err == nil &&
			nw.Ingress &&
			!command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), ingressWarning) {
			continue
		}
		if err := client.NetworkRemove(ctx, name); err != nil {
			if opts.force && errdefs.IsNotFound(err) {
				continue
			}
			fmt.Fprintf(dockerCli.Err(), "%s\n", err)
			status = 1
			continue
		}
		fmt.Fprintf(dockerCli.Out(), "%s\n", name)
	}

	if status != 0 {
		return cli.StatusError{StatusCode: status}
	}
	return nil
}
