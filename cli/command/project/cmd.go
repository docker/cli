package project

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewProjectCommand returns a cobra command struct for the `project` subcommand
func NewProjectCommand(dockerCli *command.DockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		NewInitCommand(dockerCli),
		NewJoinCommand(dockerCli),
		NewLsCommand(dockerCli),
		NewLeaveCommand(dockerCli),
		NewIDCommand(dockerCli),
	)
	return cmd
}
