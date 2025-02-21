package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/v28/cli-plugins/manager"
	"github.com/docker/cli/v28/cli-plugins/plugin"
	"github.com/docker/cli/v28/cli/command"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(func(dockerCLI command.Cli) *cobra.Command {
		goodbye := &cobra.Command{
			Use:   "goodbye",
			Short: "Say Goodbye instead of Hello",
			Run: func(cmd *cobra.Command, _ []string) {
				_, _ = fmt.Fprintln(dockerCLI.Out(), "Goodbye World!")
			},
		}
		apiversion := &cobra.Command{
			Use:   "apiversion",
			Short: "Print the API version of the server",
			RunE: func(_ *cobra.Command, _ []string) error {
				apiClient := dockerCLI.Client()
				ping, err := apiClient.Ping(context.Background())
				if err != nil {
					return err
				}
				_, _ = fmt.Println(ping.APIVersion)
				return nil
			},
		}

		exitStatus2 := &cobra.Command{
			Use:   "exitstatus2",
			Short: "Exit with status 2",
			RunE: func(_ *cobra.Command, _ []string) error {
				_, _ = fmt.Fprintln(dockerCLI.Err(), "Exiting with error status 2")
				os.Exit(2)
				return nil
			},
		}

		var (
			who, optContext string
			preRun, debug   bool
		)
		cmd := &cobra.Command{
			Use:   "helloworld",
			Short: "A basic Hello World plugin for tests",
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				if err := plugin.PersistentPreRunE(cmd, args); err != nil {
					return err
				}
				if preRun {
					_, _ = fmt.Fprintln(dockerCLI.Err(), "Plugin PersistentPreRunE called")
				}
				return nil
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				if debug {
					_, _ = fmt.Fprintln(dockerCLI.Err(), "Plugin debug mode enabled")
				}

				switch optContext {
				case "Christmas":
					_, _ = fmt.Fprintln(dockerCLI.Out(), "Merry Christmas!")
					return nil
				case "":
					// nothing
				}

				if who == "" {
					who, _ = dockerCLI.ConfigFile().PluginConfig("helloworld", "who")
				}
				if who == "" {
					who = "World"
				}

				_, _ = fmt.Fprintln(dockerCLI.Out(), "Hello", who)
				dockerCLI.ConfigFile().SetPluginConfig("helloworld", "lastwho", who)
				return dockerCLI.ConfigFile().Save()
			},
		}

		flags := cmd.Flags()
		flags.StringVar(&who, "who", "", "Who are we addressing?")
		flags.BoolVar(&preRun, "pre-run", false, "Log from prerun hook")
		// These are intended to deliberately clash with the CLIs own top
		// level arguments.
		flags.BoolVarP(&debug, "debug", "D", false, "Enable debug")
		flags.StringVarP(&optContext, "context", "c", "", "Is it Christmas?")

		cmd.AddCommand(goodbye, apiversion, exitStatus2)
		return cmd
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       "testing",
		})
}
