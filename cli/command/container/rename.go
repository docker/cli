package container

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/spf13/cobra"
)

// newRenameCommand creates a new cobra.Command for "docker container rename".
func newRenameCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename CONTAINER NEW_NAME",
		Short: "Rename a container",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRename(cmd.Context(), dockerCLI, args[0], args[1])
		},
		Annotations: map[string]string{
			"aliases": "docker container rename, docker rename",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, true),
		DisableFlagsInUseLine: true,
	}
	return cmd
}

func runRename(ctx context.Context, dockerCLI command.Cli, oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		// TODO(thaJeztah): improve validation in ContainerRename and daemon; the daemon returns an obscure error when providing whitespace-only new-name:
		// 	Error response from daemon: Error when allocating new name: Invalid container name (/ ), only [a-zA-Z0-9][a-zA-Z0-9_.-] are allowed
		return errors.New("new name cannot be blank")
	}
	if err := dockerCLI.Client().ContainerRename(ctx, oldName, newName); err != nil {
		return fmt.Errorf("failed to rename container: %w", err)
	}
	return nil
}
