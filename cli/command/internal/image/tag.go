package image

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newTagCommand)
}

type tagOptions struct {
	image string
	name  string
}

// NewTagCommand creates a new `docker tag` command
//
// This is a legacy command that can be hidden by setting the `DOCKER_HIDE_LEGACY_COMMANDS`
// environment variable.
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewTagCommand(dockerCLI command.Cli) *cobra.Command {
	return newTagCommand(dockerCLI)
}

// NewTagCommand creates a new `docker tag` command
var newTagCommand = commands.MaybeHideLegacy(func(dockerCLI command.Cli) *cobra.Command {
	var opts tagOptions

	cmd := &cobra.Command{
		Use:   "tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		Short: "Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.image = args[0]
			opts.name = args[1]
			return runTag(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image tag, docker tag",
		},
		ValidArgsFunction: completion.ImageNames(dockerCLI, 2),
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	return cmd
})

func runTag(ctx context.Context, dockerCli command.Cli, opts tagOptions) error {
	return dockerCli.Client().ImageTag(ctx, opts.image, opts.name)
}
