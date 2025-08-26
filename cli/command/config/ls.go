package config

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

// ListOptions contains options for the docker config ls command.
//
// Deprecated: this type was for internal use and will be removed in the next release.
type ListOptions struct {
	Quiet  bool
	Format string
	Filter opts.FilterOpt
}

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
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listOpts.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&listOpts.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&listOpts.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

// RunConfigList lists Swarm configs.
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunConfigList(ctx context.Context, dockerCLI command.Cli, options ListOptions) error {
	return runList(ctx, dockerCLI, listOptions{
		quiet:  options.Quiet,
		format: options.Format,
		filter: options.Filter,
	})
}

// runList lists Swarm configs.
func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()

	configs, err := apiClient.ConfigList(ctx, swarm.ConfigListOptions{Filters: options.filter.Value()})
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

	sort.Slice(configs, func(i, j int) bool {
		return sortorder.NaturalLess(configs[i].Spec.Name, configs[j].Spec.Name)
	})

	configCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
	}
	return formatWrite(configCtx, configs)
}
