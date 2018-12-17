package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Run is the top-level entry point to the CLI plugin framework. It should be called from your plugin's `main()` function.
func Run(makeCmd func(command.Cli) *cobra.Command, meta manager.Metadata) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	plugin := makeCmd(dockerCli)

	cmd := newPluginCommand(dockerCli, plugin, meta)

	if err := cmd.Execute(); err != nil {
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

// options encapsulates the ClientOptions and FlagSet constructed by
// `newPluginCommand` such that they can be finalized by our
// `PersistentPreRunE`. This is necessary because otherwise a plugin's
// own use of that hook will shadow anything we add to the top-level
// command meaning the CLI is never Initialized.
var options struct {
	init, prerun sync.Once
	opts         *cliflags.ClientOptions
	flags        *pflag.FlagSet
	dockerCli    *command.DockerCli
}

// PersistentPreRunE must be called by any plugin command (or
// subcommand) which uses the cobra `PersistentPreRun*` hook. Plugins
// which do not make use of `PersistentPreRun*` do not need to call
// this (although it remains safe to do so). Plugins are recommended
// to use `PersistenPreRunE` to enable the error to be
// returned. Should not be called outside of a commands
// PersistentPreRunE hook and must not be run unless Run has been
// called.
func PersistentPreRunE(cmd *cobra.Command, args []string) error {
	var err error
	options.prerun.Do(func() {
		if options.opts == nil || options.flags == nil || options.dockerCli == nil {
			panic("PersistentPreRunE called without Run successfully called first")
		}
		// flags must be the original top-level command flags, not cmd.Flags()
		options.opts.Common.SetDefaultOptions(options.flags)
		err = options.dockerCli.Initialize(options.opts)
	})
	return err
}

func newPluginCommand(dockerCli *command.DockerCli, plugin *cobra.Command, meta manager.Metadata) *cobra.Command {
	name := plugin.Use
	fullname := manager.NamePrefix + name

	cmd := &cobra.Command{
		Use:                   fmt.Sprintf("docker [OPTIONS] %s [ARG...]", name),
		Short:                 fullname + " is a Docker CLI plugin",
		SilenceUsage:          true,
		SilenceErrors:         true,
		TraverseChildren:      true,
		PersistentPreRunE:     PersistentPreRunE,
		DisableFlagsInUseLine: true,
	}
	opts, flags := cli.SetupPluginRootCommand(cmd)

	cmd.SetOutput(dockerCli.Out())

	cmd.AddCommand(
		plugin,
		newMetadataSubcommand(plugin, meta),
	)

	cli.DisableFlagsInUseLine(cmd)

	options.init.Do(func() {
		options.opts = opts
		options.flags = flags
		options.dockerCli = dockerCli
	})
	return cmd
}

func newMetadataSubcommand(plugin *cobra.Command, meta manager.Metadata) *cobra.Command {
	if meta.ShortDescription == "" {
		meta.ShortDescription = plugin.Short
	}
	cmd := &cobra.Command{
		Use:    manager.MetadataSubcommandName,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "     ")
			return enc.Encode(meta)
		},
	}
	return cmd
}
