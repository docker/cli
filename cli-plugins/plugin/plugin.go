package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/socket"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/cli/cli/debug"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

// PersistentPreRunE must be called by any plugin command (or
// subcommand) which uses the cobra `PersistentPreRun*` hook. Plugins
// which do not make use of `PersistentPreRun*` do not need to call
// this (although it remains safe to do so). Plugins are recommended
// to use `PersistentPreRunE` to enable the error to be
// returned. Should not be called outside of a command's
// PersistentPreRunE hook and must not be run unless Run has been
// called.
var PersistentPreRunE func(*cobra.Command, []string) error

// RunPlugin executes the specified plugin command
func RunPlugin(dockerCli *command.DockerCli, plugin *cobra.Command, meta manager.Metadata) error {
	tcmd := newPluginCommand(dockerCli, plugin, meta)

	var persistentPreRunOnce sync.Once
	PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		var err error
		persistentPreRunOnce.Do(func() {
			ctx, cancel := context.WithCancel(cmd.Context())
			cmd.SetContext(ctx)
			// Set up the context to cancel based on signalling via CLI socket.
			socket.ConnectAndWait(cancel)

			var opts []command.CLIOption
			if os.Getenv("DOCKER_CLI_PLUGIN_USE_DIAL_STDIO") != "" {
				opts = append(opts, withPluginClientConn(plugin.Name()))
			}
			opts = append(opts, command.WithEnableGlobalMeterProvider(), command.WithEnableGlobalTracerProvider())
			err = tcmd.Initialize(opts...)
			ogRunE := cmd.RunE
			if ogRunE == nil {
				ogRun := cmd.Run
				// necessary because error will always be nil here
				// see: https://github.com/golangci/golangci-lint/issues/1379
				//nolint:unparam
				ogRunE = func(cmd *cobra.Command, args []string) error {
					ogRun(cmd, args)
					return nil
				}
				cmd.Run = nil
			}
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				stopInstrumentation := dockerCli.StartInstrumentation(cmd)
				err := ogRunE(cmd, args)
				stopInstrumentation(err)
				return err
			}
		})
		return err
	}

	cmd, args, err := tcmd.HandleGlobalFlags()
	if err != nil {
		return err
	}
	// We've parsed global args already, so reset args to those
	// which remain.
	cmd.SetArgs(args)
	return cmd.Execute()
}

// Run is the top-level entry point to the CLI plugin framework. It should be called from your plugin's `main()` function.
func Run(makeCmd func(command.Cli) *cobra.Command, meta manager.Metadata) {
	otel.SetErrorHandler(debug.OTELErrorHandler)

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	plugin := makeCmd(dockerCli)

	if err := RunPlugin(dockerCli, plugin, meta); err != nil {
		if sterr, ok := err.(cli.StatusError); ok {
			if sterr.Status != "" {
				fmt.Fprintln(dockerCli.Err(), sterr.Status)
			}
			// StatusError should only be used for errors, and all errors should
			// have a non-zero exit status, so never exit with 0
			if sterr.StatusCode == 0 {
				os.Exit(1)
			}
			os.Exit(sterr.StatusCode)
		}
		fmt.Fprintln(dockerCli.Err(), err)
		os.Exit(1)
	}
}

func withPluginClientConn(name string) command.CLIOption {
	return command.WithInitializeClient(func(dockerCli *command.DockerCli) (client.APIClient, error) {
		cmd := "docker"
		if x := os.Getenv(manager.ReexecEnvvar); x != "" {
			cmd = x
		}
		var flags []string

		// Accumulate all the global arguments, that is those
		// up to (but not including) the plugin's name. This
		// ensures that `docker system dial-stdio` is
		// evaluating the same set of `--config`, `--tls*` etc
		// global options as the plugin was called with, which
		// in turn is the same as what the original docker
		// invocation was passed.
		for _, a := range os.Args[1:] {
			if a == name {
				break
			}
			flags = append(flags, a)
		}
		flags = append(flags, "system", "dial-stdio")

		helper, err := connhelper.GetCommandConnectionHelper(cmd, flags...)
		if err != nil {
			return nil, err
		}

		return client.NewClientWithOpts(client.WithDialContext(helper.Dialer))
	})
}

func newPluginCommand(dockerCli *command.DockerCli, plugin *cobra.Command, meta manager.Metadata) *cli.TopLevelCommand {
	name := plugin.Name()
	fullname := manager.NamePrefix + name

	cmd := &cobra.Command{
		Use:           fmt.Sprintf("docker [OPTIONS] %s [ARG...]", name),
		Short:         fullname + " is a Docker CLI plugin",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// We can't use this as the hook directly since it is initialised later (in runPlugin)
			return PersistentPreRunE(cmd, args)
		},
		TraverseChildren:      true,
		DisableFlagsInUseLine: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			HiddenDefaultCmd:    true,
			DisableDescriptions: true,
		},
	}
	opts, _ := cli.SetupPluginRootCommand(cmd)

	cmd.SetIn(dockerCli.In())
	cmd.SetOut(dockerCli.Out())
	cmd.SetErr(dockerCli.Err())

	cmd.AddCommand(
		plugin,
		newMetadataSubcommand(plugin, meta),
	)

	cli.DisableFlagsInUseLine(cmd)

	return cli.NewTopLevelCommand(cmd, dockerCli, opts, cmd.Flags())
}

func newMetadataSubcommand(plugin *cobra.Command, meta manager.Metadata) *cobra.Command {
	if meta.ShortDescription == "" {
		meta.ShortDescription = plugin.Short
	}
	cmd := &cobra.Command{
		Use:    manager.MetadataSubcommandName,
		Hidden: true,
		// Suppress the global/parent PersistentPreRunE, which
		// needlessly initializes the client and tries to
		// connect to the daemon.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "     ")
			return enc.Encode(meta)
		},
	}
	return cmd
}

// RunningStandalone tells a CLI plugin it is run standalone by direct execution
func RunningStandalone() bool {
	if os.Getenv(manager.ReexecEnvvar) != "" {
		return false
	}
	return len(os.Args) < 2 || os.Args[1] != manager.MetadataSubcommandName
}
