package manager

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
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

	// CommandAnnotationPluginCommandPath is added to overwrite the
	// command path for a plugin invocation.
	CommandAnnotationPluginCommandPath = "com.docker.cli.plugin.command_path"
)

var pluginCommandStubsOnce sync.Once

// AddPluginCommandStubs adds a stub cobra.Commands for each valid and invalid
// plugin. The command stubs will have several annotations added, see
// `CommandAnnotationPlugin*`.
func AddPluginCommandStubs(dockerCli command.Cli, rootCmd *cobra.Command) (err error) {
	pluginCommandStubsOnce.Do(func() {
		var plugins []Plugin
		plugins, err = ListPlugins(dockerCli, rootCmd)
		if err != nil {
			return
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
					perr := flags.Parse(args)
					if perr != nil {
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
					runCommand, runErr := PluginRunCommand(dockerCli, p.Name, cmd)
					if runErr != nil {
						return nil, cobra.ShellCompDirectiveError
					}
					runErr = runCommand.Run()
					if runErr == nil {
						os.Exit(0) // plugin already rendered complete data
					}
					return nil, cobra.ShellCompDirectiveError
				},
			})
		}
	})
	return err
}

const (
	dockerCliAttributePrefix = attribute.Key("docker.cli")

	cobraCommandPath = attribute.Key("cobra.command_path")
)

func getPluginResourceAttributes(cmd *cobra.Command, plugin Plugin) attribute.Set {
	commandPath := cmd.Annotations[CommandAnnotationPluginCommandPath]
	if commandPath == "" {
		commandPath = fmt.Sprintf("%s %s", cmd.CommandPath(), plugin.Name)
	}

	attrSet := attribute.NewSet(
		cobraCommandPath.String(commandPath),
	)

	kvs := make([]attribute.KeyValue, 0, attrSet.Len())
	for iter := attrSet.Iter(); iter.Next(); {
		attr := iter.Attribute()
		kvs = append(kvs, attribute.KeyValue{
			Key:   dockerCliAttributePrefix + "." + attr.Key,
			Value: attr.Value,
		})
	}
	return attribute.NewSet(kvs...)
}

func appendPluginResourceAttributesEnvvar(env []string, cmd *cobra.Command, plugin Plugin) []string {
	if attrs := getPluginResourceAttributes(cmd, plugin); attrs.Len() > 0 {
		// values in environment variables need to be in baggage format
		// otel/baggage package can be used after update to v1.22, currently it encodes incorrectly
		attrsSlice := make([]string, attrs.Len())
		for iter := attrs.Iter(); iter.Next(); {
			i, v := iter.IndexedAttribute()
			attrsSlice[i] = string(v.Key) + "=" + url.PathEscape(v.Value.AsString())
		}
		env = append(env, ResourceAttributesEnvvar+"="+strings.Join(attrsSlice, ","))
	}
	return env
}
