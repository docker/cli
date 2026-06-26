package plugin

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newDisableCommand(dockerCLI command.Cli) *cobra.Command {
	var opts client.PluginDisableOptions

	cmd := &cobra.Command{
		Use:   "disable [OPTIONS] PLUGIN",
		Short: "Disable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if _, err := dockerCLI.Client().PluginDisable(cmd.Context(), name, opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(dockerCLI.Out(), name)
			return nil
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateEnabled),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force the disable of an active plugin")
	return cmd
}
