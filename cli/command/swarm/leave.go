package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type leaveOptions struct {
	force bool
}

func newLeaveCommand(dockerCli command.Cli) *cobra.Command {
	opts := leaveOptions{}

	cmd := &cobra.Command{
		Use:   "leave [OPTIONS]",
		Short: "Leave the swarm",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLeave(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "active",
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force this node to leave the swarm, ignoring warnings")
	return cmd
}

func runLeave(ctx context.Context, dockerCLI command.Cli, opts leaveOptions) error {
	apiClient := dockerCLI.Client()

	if err := apiClient.SwarmLeave(ctx, opts.force); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), "Node left the swarm.")
	return nil
}
