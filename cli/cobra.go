package cli

import (
	"fmt"
	"strings"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	cliconfig "github.com/docker/cli/cli/config"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// setupCommonRootCommand contains the setup common to
// SetupRootCommand and SetupPluginRootCommand.
func setupCommonRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet, *cobra.Command) {
	opts := cliflags.NewClientOptions()
	flags := rootCmd.Flags()

	flags.StringVar(&opts.ConfigDir, "config", cliconfig.Dir(), "Location of client config files")
	opts.Common.InstallFlags(flags)

	cobra.AddTemplateFunc("add", func(a, b int) int { return a + b })
	cobra.AddTemplateFunc("hasSubCommands", hasSubCommands)
	cobra.AddTemplateFunc("hasManagementSubCommands", hasManagementSubCommands)
	cobra.AddTemplateFunc("hasInvalidPlugins", hasInvalidPlugins)
	cobra.AddTemplateFunc("operationSubCommands", operationSubCommands)
	cobra.AddTemplateFunc("managementSubCommands", managementSubCommands)
	cobra.AddTemplateFunc("invalidPlugins", invalidPlugins)
	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cobra.AddTemplateFunc("vendorAndVersion", vendorAndVersion)
	cobra.AddTemplateFunc("invalidPluginReason", invalidPluginReason)
	cobra.AddTemplateFunc("isPlugin", isPlugin)
	cobra.AddTemplateFunc("decoratedName", decoratedName)

	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetFlagErrorFunc(FlagErrorFunc)
	rootCmd.SetHelpCommand(helpCommand)

	return opts, flags, helpCommand
}

// SetupRootCommand sets default usage, help, and error handling for the
// root command.
func SetupRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet, *cobra.Command) {
	opts, flags, helpCmd := setupCommonRootCommand(rootCmd)

	rootCmd.SetVersionTemplate("Docker version {{.Version}}\n")

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")
	rootCmd.PersistentFlags().Lookup("help").Hidden = true

	return opts, flags, helpCmd
}

// SetupPluginRootCommand sets default usage, help and error handling for a plugin root command.
func SetupPluginRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet) {
	opts, flags, _ := setupCommonRootCommand(rootCmd)

	rootCmd.PersistentFlags().BoolP("help", "", false, "Print usage")
	rootCmd.PersistentFlags().Lookup("help").Hidden = true

	return opts, flags
}

// FlagErrorFunc prints an error message which matches the format of the
// docker/cli/cli error messages
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	usage := ""
	if cmd.HasSubCommands() {
		usage = "\n\n" + cmd.UsageString()
	}
	return StatusError{
		Status:     fmt.Sprintf("%s\nSee '%s --help'.%s", err, cmd.CommandPath(), usage),
		StatusCode: 125,
	}
}

// VisitAll will traverse all commands from the root.
// This is different from the VisitAll of cobra.Command where only parents
// are checked.
func VisitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, cmd := range root.Commands() {
		VisitAll(cmd, fn)
	}
	fn(root)
}

// DisableFlagsInUseLine sets the DisableFlagsInUseLine flag on all
// commands within the tree rooted at cmd.
func DisableFlagsInUseLine(cmd *cobra.Command) {
	VisitAll(cmd, func(ccmd *cobra.Command) {
		// do not add a `[flags]` to the end of the usage line.
		ccmd.DisableFlagsInUseLine = true
	})
}

var helpCommand = &cobra.Command{
	Use:               "help [command]",
	Short:             "Help about the command",
	PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	RunE: func(c *cobra.Command, args []string) error {
		cmd, args, e := c.Root().Find(args)
		if cmd == nil || e != nil || len(args) > 0 {
			return errors.Errorf("unknown help topic: %v", strings.Join(args, " "))
		}

		helpFunc := cmd.HelpFunc()
		helpFunc(cmd, args)
		return nil
	},
}

func isPlugin(cmd *cobra.Command) bool {
	return cmd.Annotations[pluginmanager.CommandAnnotationPlugin] == "true"
}

func hasSubCommands(cmd *cobra.Command) bool {
	return len(operationSubCommands(cmd)) > 0
}

func hasManagementSubCommands(cmd *cobra.Command) bool {
	return len(managementSubCommands(cmd)) > 0
}

func hasInvalidPlugins(cmd *cobra.Command) bool {
	return len(invalidPlugins(cmd)) > 0
}

func operationSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isPlugin(sub) {
			continue
		}
		if sub.IsAvailableCommand() && !sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func wrappedFlagUsages(cmd *cobra.Command) string {
	width := 80
	if ws, err := term.GetWinsize(0); err == nil {
		width = int(ws.Width)
	}
	return cmd.Flags().FlagUsagesWrapped(width - 1)
}

func decoratedName(cmd *cobra.Command) string {
	decoration := " "
	if isPlugin(cmd) {
		decoration = "*"
	}
	return cmd.Name() + decoration
}

func vendorAndVersion(cmd *cobra.Command) string {
	if vendor, ok := cmd.Annotations[pluginmanager.CommandAnnotationPluginVendor]; ok && isPlugin(cmd) {
		version := ""
		if v, ok := cmd.Annotations[pluginmanager.CommandAnnotationPluginVersion]; ok && v != "" {
			version = ", " + v
		}
		return fmt.Sprintf("(%s%s)", vendor, version)
	}
	return ""
}

func managementSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isPlugin(sub) {
			if invalidPluginReason(sub) == "" {
				cmds = append(cmds, sub)
			}
			continue
		}
		if sub.IsAvailableCommand() && sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func invalidPlugins(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if !isPlugin(sub) {
			continue
		}
		if invalidPluginReason(sub) != "" {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func invalidPluginReason(cmd *cobra.Command) string {
	return cmd.Annotations[pluginmanager.CommandAnnotationPluginInvalid]
}

var usageTemplate = `Usage:

{{- if not .HasSubCommands}}	{{.UseLine}}{{end}}
{{- if .HasSubCommands}}	{{ .CommandPath}}{{- if .HasAvailableFlags}} [OPTIONS]{{end}} COMMAND{{end}}

{{if ne .Long ""}}{{ .Long | trim }}{{ else }}{{ .Short | trim }}{{end}}

{{- if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}

{{- end}}
{{- if .HasExample}}

Examples:
{{ .Example }}

{{- end}}
{{- if .HasAvailableFlags}}

Options:
{{ wrappedFlagUsages . | trimRightSpace}}

{{- end}}
{{- if hasManagementSubCommands . }}

Management Commands:

{{- range managementSubCommands . }}
  {{rpad (decoratedName .) (add .NamePadding 1)}}{{.Short}}{{ if isPlugin .}} {{vendorAndVersion .}}{{ end}}
{{- end}}

{{- end}}
{{- if hasSubCommands .}}

Commands:

{{- range operationSubCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}
{{- end}}

{{- if hasInvalidPlugins . }}

Invalid Plugins:

{{- range invalidPlugins . }}
  {{rpad .Name .NamePadding }} {{invalidPluginReason .}}
{{- end}}

{{- end}}

{{- if .HasSubCommands }}

Run '{{.CommandPath}} COMMAND --help' for more information on a command.
{{- end}}
`

var helpTemplate = `
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
