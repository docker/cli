package main

import (
	"fmt"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		goodbye := &cobra.Command{
			Use:   "goodbye",
			Short: "Say Goodbye instead of Hello",
			Run: func(cmd *cobra.Command, _ []string) {
				fmt.Fprintln(dockerCli.Out(), "Goodbye World!")
			},
		}

		cmd := &cobra.Command{
			Use:   "helloworld",
			Short: "A basic Hello World plugin for tests",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Fprintln(dockerCli.Out(), "Hello World!")
			},
		}

		cmd.AddCommand(goodbye)
		return cmd
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       "testing",
		})
}
