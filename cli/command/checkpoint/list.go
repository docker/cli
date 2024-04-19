package checkpoint

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/checkpoint"
	"github.com/spf13/cobra"
)

type listOptions struct {
	checkpointDir string
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS] CONTAINER",
		Aliases: []string{"list"},
		Short:   "List checkpoints for a container",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCli, args[0], opts)
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false),
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")

	return cmd
}

func runList(ctx context.Context, dockerCli command.Cli, container string, opts listOptions) error {
	checkpoints, err := dockerCli.Client().CheckpointList(ctx, container, checkpoint.ListOptions{
		CheckpointDir: opts.checkpointDir,
	})
	if err != nil {
		return err
	}

	cpCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(formatter.TableFormatKey),
	}
	return FormatWrite(cpCtx, checkpoints)
}
