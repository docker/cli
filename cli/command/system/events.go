package system

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/events"
	"github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

// NewEventsCommand creates a new cobra.Command for `docker events`
func NewEventsCommand(dockerCli command.Cli) *cobra.Command {
	options := events.Options{Filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "events [OPTIONS]",
		Short: "Get real time events from the server",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return events.Run(dockerCli, &options)
		},
		Annotations: map[string]string{
			"aliases": "docker system events, docker events",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	events.InstallFlags(flags, &options)
	return cmd
}
