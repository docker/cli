package container

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// newRenameCommand creates a new cobra.Command for "docker container rename".
func newRenameCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename CONTAINER NEW_NAME",
		Short: "Rename a container",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			_, err := dockerCLI.Client().ContainerRename(cmd.Context(), oldName, client.ContainerRenameOptions{
				NewName: newName,
			})
			if err != nil {
				return fmt.Errorf("failed to rename container: %w", err)
			}
			return nil
		},
		Annotations: map[string]string{
			"aliases": "docker container rename, docker rename",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, true),
		DisableFlagsInUseLine: true,
	}
	return cmd
}
