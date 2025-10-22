package config

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

// listOptions contains options for the docker config ls command.
type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newConfigListCommand(dockerCLI command.Cli) *cobra.Command {
	listOpts := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List configs",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, listOpts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listOpts.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&listOpts.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&listOpts.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

// runList lists Swarm configs.
func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()

	res, err := apiClient.ConfigList(ctx, client.ConfigListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ConfigFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().ConfigFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(res.Items, func(i, j int) bool {
		return sortorder.NaturalLess(res.Items[i].Spec.Name, res.Items[j].Spec.Name)
	})

	configCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
	}
	return formatWrite(configCtx, res)
}
