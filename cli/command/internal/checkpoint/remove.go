package checkpoint

import (
	"context"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/moby/moby/api/types/checkpoint"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	checkpointDir string
}

func newRemoveCommand(dockerCLI cli.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] CONTAINER CHECKPOINT",
		Aliases: []string{"remove"},
		Short:   "Remove a checkpoint",
		Args:    cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args[0], args[1], opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")

	return cmd
}

func runRemove(ctx context.Context, dockerCLI cli.Cli, container string, checkpointID string, opts removeOptions) error {
	return dockerCLI.Client().CheckpointDelete(ctx, container, checkpoint.DeleteOptions{
		CheckpointID:  checkpointID,
		CheckpointDir: opts.checkpointDir,
	})
}
