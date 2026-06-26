package container

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// newDiffCommand creates a new cobra.Command for `docker diff`
func newDiffCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "diff CONTAINER",
		Short: "Inspect changes to files or directories on a container's filesystem",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(cmd.Context(), dockerCLI, args[0])
		},
		Annotations: map[string]string{
			"aliases": "docker container diff, docker diff",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}
}

func runDiff(ctx context.Context, dockerCLI command.Cli, containerID string) error {
	res, err := dockerCLI.Client().ContainerDiff(ctx, containerID, client.ContainerDiffOptions{})
	if err != nil {
		return err
	}
	diffCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newDiffFormat("{{.Type}} {{.Path}}"),
	}
	return diffFormatWrite(diffCtx, res)
}
