package main

import (
	"context"
	"fmt"
	"os"

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

		exitStatus2 := &cobra.Command{
			Use:   "exitstatus2",
			Short: "Exit with status 2",
			RunE: func(_ *cobra.Command, _ []string) error {
				fmt.Fprintln(dockerCli.Err(), "Exiting with error status 2")
				os.Exit(2)
				return nil
			},
		}

		var (
			who, context  string
			preRun, debug bool
		)
		cmd := &cobra.Command{
			Use:   "helloworld",
			Short: "A basic Hello World plugin for tests",
			PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
				if preRun {
					fmt.Fprintf(dockerCli.Err(), "Plugin PersistentPreRunE called")
				}
				return nil
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				if debug {
					fmt.Fprintf(dockerCli.Err(), "Plugin debug mode enabled")
				}

				switch context {
				case "Christmas":
					fmt.Fprintf(dockerCli.Out(), "Merry Christmas!\n")
					return nil
				case "":
					// nothing
				}

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
		flags.BoolVar(&preRun, "pre-run", false, "Log from prerun hook")
		// These are intended to deliberately clash with the CLIs own top
		// level arguments.
		flags.BoolVarP(&debug, "debug", "D", false, "Enable debug")
		flags.StringVarP(&context, "context", "c", "", "Is it Christmas?")

		cmd.AddCommand(goodbye, apiversion, exitStatus2)
		return cmd
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       "testing",
		})
}
