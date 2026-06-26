package checkpoint

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type listOptions struct {
	checkpointDir string
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS] CONTAINER",
		Aliases: []string{"list"},
		Short:   "List checkpoints for a container",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, args[0], opts)
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")

	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, container string, opts listOptions) error {
	checkpoints, err := dockerCLI.Client().CheckpointList(ctx, container, client.CheckpointListOptions{
		CheckpointDir: opts.checkpointDir,
	})
	if err != nil {
		return err
	}

	cpCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(formatter.TableFormatKey),
	}
	return formatWrite(cpCtx, checkpoints.Items)
}
