package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// PersistentPreRunE must be called by any plugin command (or
// subcommand) which uses the cobra `PersistentPreRun*` hook. Plugins
// which do not make use of `PersistentPreRun*` do not need to call
// this (although it remains safe to do so). Plugins are recommended
// to use `PersistenPreRunE` to enable the error to be
// returned. Should not be called outside of a command's
// PersistentPreRunE hook and must not be run unless Run has been
// called.
var PersistentPreRunE func(*cobra.Command, []string) error

func runPlugin(dockerCli *command.DockerCli, plugin *cobra.Command, meta manager.Metadata) error {
	tcmd := newPluginCommand(dockerCli, plugin, meta)

	var persistentPreRunOnce sync.Once
	PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		var err error
		persistentPreRunOnce.Do(func() {
			var opts []command.InitializeOpt
			if os.Getenv("DOCKER_CLI_PLUGIN_USE_DIAL_STDIO") != "" {
				opts = append(opts, withPluginClientConn(plugin.Name()))
			}
			err = tcmd.Initialize(opts...)
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
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	plugin := makeCmd(dockerCli)

	if err := runPlugin(dockerCli, plugin, meta); err != nil {
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

func withPluginClientConn(name string) command.InitializeOpt {
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
	}
	opts, flags := cli.SetupPluginRootCommand(cmd)

	cmd.SetOutput(dockerCli.Out())

	cmd.AddCommand(
		plugin,
		newMetadataSubcommand(plugin, meta),
	)

	cli.DisableFlagsInUseLine(cmd)

	return cli.NewTopLevelCommand(cmd, dockerCli, opts, flags)
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
