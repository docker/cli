package node

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/swarm"
	"github.com/spf13/cobra"
)

func newPromoteCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "promote NODE [NODE...]",
		Short: "Promote one or more nodes to manager in the swarm",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd.Context(), dockerCli, args)
		},
	}
}

func runPromote(ctx context.Context, dockerCli command.Cli, nodes []string) error {
	promote := func(node *swarm.Node) error {
		if node.Spec.Role == swarm.NodeRoleManager {
			fmt.Fprintf(dockerCli.Out(), "Node %s is already a manager.\n", node.ID)
			return errNoRoleChange
		}
		node.Spec.Role = swarm.NodeRoleManager
		return nil
	}
	success := func(nodeID string) {
		fmt.Fprintf(dockerCli.Out(), "Node %s promoted to a manager in the swarm.\n", nodeID)
	}
	return updateNodes(ctx, dockerCli, nodes, promote, success)
}
