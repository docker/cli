package plugin

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
)

type listOptions struct {
	quiet   bool
	noTrunc bool
	format  string
	filter  opts.FilterOpt
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Short:   "List plugins",
		Aliases: []string{"list"},
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display plugin IDs")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.StringVar(&options.format, "format", "", "Pretty-print plugins using a Go template")
	flags.VarP(&options.filter, "filter", "f", "Provide filter values (e.g. 'enabled=true')")

	return cmd
}

func runList(dockerCli command.Cli, options listOptions) error {
	plugins, err := dockerCli.Client().PluginList(context.Background(), options.filter.Value())
	if err != nil {
		return err
	}

	sort.Slice(plugins, func(i, j int) bool {
		return sortorder.NaturalLess(plugins[i].Name, plugins[j].Name)
	})

	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().PluginsFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().PluginsFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	pluginsCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(format, options.quiet),
		Trunc:  !options.noTrunc,
	}
	return FormatWrite(pluginsCtx, plugins)
}
