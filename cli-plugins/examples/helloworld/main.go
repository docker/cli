package main

import (
	"context"
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
		apiversion := &cobra.Command{
			Use:   "apiversion",
			Short: "Print the API version of the server",
			RunE: func(_ *cobra.Command, _ []string) error {
				cli := dockerCli.Client()
				ping, err := cli.Ping(context.Background())
				if err != nil {
					return err
				}
				fmt.Println(ping.APIVersion)
				return nil
			},
		}

		var who string
		cmd := &cobra.Command{
			Use:   "helloworld",
			Short: "A basic Hello World plugin for tests",
			// This is redundant but included to exercise
			// the path where a plugin overrides this
			// hook.
			PersistentPreRunE: plugin.PersistentPreRunE,
			RunE: func(cmd *cobra.Command, args []string) error {
				if who == "" {
					who, _ = dockerCli.ConfigFile().PluginConfig("helloworld", "who")
				}
				if who == "" {
					who = "World"
				}

				fmt.Fprintf(dockerCli.Out(), "Hello %s!\n", who)
				dockerCli.ConfigFile().SetPluginConfig("helloworld", "lastwho", who)
				return dockerCli.ConfigFile().Save()
			},
		}
		flags := cmd.Flags()
		flags.StringVar(&who, "who", "", "Who are we addressing?")

		cmd.AddCommand(goodbye, apiversion)
		return cmd
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       "testing",
		})
}
