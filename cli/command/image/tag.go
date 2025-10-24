package image

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// newTagCommand creates a new "docker image tag" command.
func newTagCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		Short: "Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := dockerCLI.Client().ImageTag(cmd.Context(), client.ImageTagOptions{
				Source: args[0],
				Target: args[1],
			})
			return err
		},
		Annotations: map[string]string{
			"aliases": "docker image tag, docker tag",
		},
		ValidArgsFunction:     completion.ImageNames(dockerCLI, 2),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	return cmd
}
