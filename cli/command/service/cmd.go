package service

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewServiceCommand returns a cobra command for `service` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewServiceCommand(dockerCLI command.Cli) *cobra.Command {
	return newServiceCommand(dockerCLI)
}

// newServiceCommand returns a cobra command for `service` subcommands
func newServiceCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage Swarm services",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newCreateCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newPsCommand(dockerCLI),
		newListCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newScaleCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
		newLogsCommand(dockerCLI),
		newRollbackCommand(dockerCLI),
	)
	return cmd
}
