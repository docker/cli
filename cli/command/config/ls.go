// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package config

import (
	"context"
	"slices"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/api/types/swarm"
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

	slices.SortFunc(res.Items, func(a, b swarm.Config) int {
		return sortorder.NaturalCompare(a.Spec.Name, b.Spec.Name)
	})

	configCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
	}
	return formatWrite(configCtx, res)
}
