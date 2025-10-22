package network

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
		Aliases: []string{"list"},
		Short:   "List networks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display network IDs")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Do not truncate the output")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", `Provide filter values (e.g. "driver=bridge")`)

	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()
	res, err := apiClient.NetworkList(ctx, client.NetworkListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().NetworksFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().NetworksFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(res.Items, func(i, j int) bool {
		return sortorder.NaturalLess(res.Items[i].Name, res.Items[j].Name)
	})

	networksCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
		Trunc:  !options.noTrunc,
	}
	return formatWrite(networksCtx, res)
}
