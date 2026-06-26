package network

import (
	"context"
	"fmt"
	"strconv"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/internal/prompt"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool
}

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm NETWORK [NETWORK...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more networks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args, &opts)
		},
		ValidArgsFunction:     completion.NetworkNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Do not error if the network does not exist")
	return cmd
}

const ingressWarning = "WARNING! Before removing the routing-mesh network, " +
	"make sure all the nodes in your swarm run the same docker engine version. " +
	"Otherwise, removal may not be effective and functionality of newly create " +
	"ingress networks will be impaired.\nAre you sure you want to continue?"

func runRemove(ctx context.Context, dockerCLI command.Cli, networks []string, opts *removeOptions) error {
	apiClient := dockerCLI.Client()

	status := 0

	for _, name := range networks {
		res, err := apiClient.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
		if err == nil && res.Network.Ingress {
			r, err := prompt.Confirm(ctx, dockerCLI.In(), dockerCLI.Out(), ingressWarning)
			if err != nil {
				return err
			}
			if !r {
				continue
			}
		}
		_, err = apiClient.NetworkRemove(ctx, name, client.NetworkRemoveOptions{})
		if err != nil {
			if opts.force && errdefs.IsNotFound(err) {
				continue
			}
			_, _ = fmt.Fprintln(dockerCLI.Err(), err)
			status = 1
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}

	if status != 0 {
		return cli.StatusError{StatusCode: status, Status: "exit status " + strconv.Itoa(status)}
	}
	return nil
}
