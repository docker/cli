package manager

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

const (
	// CommandAnnotationPlugin is added to every stub command added by
	// AddPluginCommandStubs with the value "true" and so can be
	// used to distinguish plugin stubs from regular commands.
	CommandAnnotationPlugin = "com.docker.cli.plugin"

	// CommandAnnotationPluginVendor is added to every stub command
	// added by AddPluginCommandStubs and contains the vendor of
	// that plugin.
	CommandAnnotationPluginVendor = "com.docker.cli.plugin.vendor"

	// CommandAnnotationPluginVersion is added to every stub command
	// added by AddPluginCommandStubs and contains the version of
	// that plugin.
	CommandAnnotationPluginVersion = "com.docker.cli.plugin.version"

	// CommandAnnotationPluginInvalid is added to any stub command
	// added by AddPluginCommandStubs for an invalid command (that
	// is, one which failed it's candidate test) and contains the
	// reason for the failure.
	CommandAnnotationPluginInvalid = "com.docker.cli.plugin-invalid"
)

// AddPluginCommandStubs adds a stub cobra.Commands for each valid and invalid
// plugin. The command stubs will have several annotations added, see
// `CommandAnnotationPlugin*`.
func AddPluginCommandStubs(dockerCli command.Cli, rootCmd *cobra.Command) error {
	plugins, err := ListPlugins(dockerCli, rootCmd)
	if err != nil {
		return err
	}
	for _, p := range plugins {
		p := p
		vendor := p.Vendor
		if vendor == "" {
			vendor = "unknown"
		}
		annotations := map[string]string{
			CommandAnnotationPlugin:        "true",
			CommandAnnotationPluginVendor:  vendor,
			CommandAnnotationPluginVersion: p.Version,
		}
		if p.Err != nil {
			annotations[CommandAnnotationPluginInvalid] = p.Err.Error()
		}
		rootCmd.AddCommand(&cobra.Command{
			Use:                p.Name,
			Short:              p.ShortDescription,
			Run:                func(_ *cobra.Command, _ []string) {},
			Annotations:        annotations,
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				flags := rootCmd.PersistentFlags()
				flags.SetOutput(nil)
				err := flags.Parse(args)
				if err != nil {
					return err
				}
				if flags.Changed("help") {
					cmd.HelpFunc()(rootCmd, args)
					return nil
				}
				return fmt.Errorf("docker: '%s' is not a docker command.\nSee 'docker --help'", cmd.Name())
			},
			ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				// Delegate completion to plugin
				cargs := []string{p.Path, cobra.ShellCompRequestCmd, p.Name}
				cargs = append(cargs, args...)
				cargs = append(cargs, toComplete)
				os.Args = cargs
				runCommand, err := PluginRunCommand(dockerCli, p.Name, cmd)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				err = runCommand.Run()
				if err == nil {
					os.Exit(0) // plugin already rendered complete data
				}
				return nil, cobra.ShellCompDirectiveError
			},
		})
	}
	return nil
}
