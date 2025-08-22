package plugin

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

func newDisableCommand(dockerCLI command.Cli) *cobra.Command {
	var opts types.PluginDisableOptions

	cmd := &cobra.Command{
		Use:   "disable [OPTIONS] PLUGIN",
		Short: "Disable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := dockerCLI.Client().PluginDisable(cmd.Context(), name, opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(dockerCLI.Out(), name)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force the disable of an active plugin")
	return cmd
}
