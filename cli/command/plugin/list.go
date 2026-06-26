package plugin

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet   bool
	noTrunc bool
	format  string
	filter  opts.FilterOpt
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Short:   "List plugins",
		Aliases: []string{"list"},
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display plugin IDs")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", `Provide filter values (e.g. "enabled=true")`)

	return cmd
}

func runList(ctx context.Context, dockerCli command.Cli, options listOptions) error {
	resp, err := dockerCli.Client().PluginList(ctx, client.PluginListOptions{
		Filters: options.filter.Value(),
	})
	if err != nil {
		return err
	}

	sort.Slice(resp.Items, func(i, j int) bool {
		return sortorder.NaturalLess(resp.Items[i].Name, resp.Items[j].Name)
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
		Format: newFormat(format, options.quiet),
		Trunc:  !options.noTrunc,
	}
	return formatWrite(pluginsCtx, resp)
}
