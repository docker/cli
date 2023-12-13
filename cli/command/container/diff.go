package container

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type diffOptions struct {
	container string
}

// NewDiffCommand creates a new cobra.Command for `docker diff`
func NewDiffCommand(dockerCli command.Cli) *cobra.Command {
	var opts diffOptions

	return &cobra.Command{
		Use:   "diff CONTAINER",
		Short: "Inspect changes to files or directories on a container's filesystem",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			return runDiff(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container diff, docker diff",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false),
	}
}

func runDiff(ctx context.Context, dockerCli command.Cli, opts *diffOptions) error {
	if opts.container == "" {
		return errors.New("Container name cannot be empty")
	}
	changes, err := dockerCli.Client().ContainerDiff(ctx, opts.container)
	if err != nil {
		return err
	}
	diffCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewDiffFormat("{{.Type}} {{.Path}}"),
	}
	return DiffFormatWrite(diffCtx, changes)
}
