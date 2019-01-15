package context

import (
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func newUseCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use CONTEXT",
		Short: "Set the current docker context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := validateContextName(name); err != nil && name != "default" {
				return err
			}
			if _, err := dockerCli.ContextStore().GetContextMetadata(name); err != nil && name != "default" {
				return err
			}
			configValue := name
			if configValue == "default" {
				configValue = ""
			}
			dockerConfig := dockerCli.ConfigFile()
			dockerConfig.CurrentContext = configValue
			if err := dockerConfig.Save(); err != nil {
				return err
			}
			fmt.Fprintln(dockerCli.Out(), name)
			fmt.Fprintf(dockerCli.Err(), "Current context is now %q\n", name)
			return nil
		},
	}
	return cmd
}
