package checkpoint

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/checkpoint"
	"github.com/spf13/cobra"
)

type createOptions struct {
	container     string
	checkpoint    string
	checkpointDir string
	leaveRunning  bool
}

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONTAINER CHECKPOINT",
		Short: "Create a checkpoint from a running container",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			opts.checkpoint = args[1]
			return runCreate(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.leaveRunning, "leave-running", false, "Leave the container running after checkpoint")
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")

	return cmd
}

func runCreate(ctx context.Context, dockerCli command.Cli, opts createOptions) error {
	err := dockerCli.Client().CheckpointCreate(ctx, opts.container, checkpoint.CreateOptions{
		CheckpointID:  opts.checkpoint,
		CheckpointDir: opts.checkpointDir,
		Exit:          !opts.leaveRunning,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", opts.checkpoint)
	return nil
}
