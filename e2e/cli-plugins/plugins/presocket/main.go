package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(RootCmd, manager.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "Docker Inc.",
		Version:       "test",
	})
}

func RootCmd(dockerCli command.Cli) *cobra.Command {
	cmd := cobra.Command{
		Use:   "presocket",
		Short: "testing plugin that does not connect to the socket",
		// override PersistentPreRunE so that the plugin default
		// PersistentPreRunE doesn't run, simulating a plugin built
		// with a pre-socket-communication version of the CLI
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "test-no-socket",
		Short: "test command that runs until it receives a SIGINT",
		RunE: func(cmd *cobra.Command, args []string) error {
			go func() {
				<-cmd.Context().Done()
				fmt.Fprintln(dockerCli.Out(), "context cancelled")
				os.Exit(2)
			}()
			signalCh := make(chan os.Signal, 10)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				for range signalCh {
					fmt.Fprintln(dockerCli.Out(), "received SIGINT")
				}
			}()
			<-time.After(3 * time.Second)
			fmt.Fprintln(dockerCli.Err(), "exit after 3 seconds")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "test-socket",
		Short: "test command that runs until it receives a SIGINT",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return plugin.PersistentPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			go func() {
				<-cmd.Context().Done()
				fmt.Fprintln(dockerCli.Out(), "context cancelled")
				os.Exit(2)
			}()
			signalCh := make(chan os.Signal, 10)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				for range signalCh {
					fmt.Fprintln(dockerCli.Out(), "received SIGINT")
				}
			}()
			<-time.After(3 * time.Second)
			fmt.Fprintln(dockerCli.Err(), "exit after 3 seconds")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "test-socket-ignore-context",
		Short: "test command that runs until it receives a SIGINT",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return plugin.PersistentPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			signalCh := make(chan os.Signal, 10)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				for range signalCh {
					fmt.Fprintln(dockerCli.Out(), "received SIGINT")
				}
			}()
			<-time.After(3 * time.Second)
			fmt.Fprintln(dockerCli.Err(), "exit after 3 seconds")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "tty",
		Short: "test command that attempts to read from the TTY",
		RunE: func(cmd *cobra.Command, args []string) error {
			done := make(chan struct{})
			go func() {
				b := make([]byte, 1)
				_, _ = dockerCli.In().Read(b)
				done <- struct{}{}
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				fmt.Fprint(dockerCli.Err(), "timeout after 2 seconds")
			}
			return nil
		},
	})

	return &cmd
}
