package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newRenameCommand)
}

type renameOptions struct {
	oldName string
	newName string
}

// NewRenameCommand creates a new cobra.Command for `docker rename`
//
// This is a legacy command that can be hidden by setting the `DOCKER_HIDE_LEGACY_COMMANDS`
// environment variable.
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewRenameCommand(dockerCli command.Cli) *cobra.Command {
	return newRenameCommand(dockerCli)
}

// newRenameCommand creates a new cobra.Command for `docker rename`
func newRenameCommand(dockerCli command.Cli) *cobra.Command {
	var opts renameOptions

	cmd := &cobra.Command{
		Use:   "rename CONTAINER NEW_NAME",
		Short: "Rename a container",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.oldName = args[0]
			opts.newName = args[1]
			return runRename(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container rename, docker rename",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, true),
	}
	return cmd
}

func runRename(ctx context.Context, dockerCli command.Cli, opts *renameOptions) error {
	oldName := strings.TrimSpace(opts.oldName)
	newName := strings.TrimSpace(opts.newName)

	if oldName == "" || newName == "" {
		return errors.New("Error: Neither old nor new names may be empty")
	}

	if err := dockerCli.Client().ContainerRename(ctx, oldName, newName); err != nil {
		fmt.Fprintln(dockerCli.Err(), err)
		return errors.Errorf("Error: failed to rename container named %s", oldName)
	}
	return nil
}
