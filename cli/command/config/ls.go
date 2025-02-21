package config

import (
	"context"
	"sort"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/completion"
	"github.com/docker/cli/v28/cli/command/formatter"
	flagsHelper "github.com/docker/cli/v28/cli/flags"
	"github.com/docker/cli/v28/opts"
	"github.com/docker/docker/api/types"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

// ListOptions contains options for the docker config ls command.
type ListOptions struct {
	Quiet  bool
	Format string
	Filter opts.FilterOpt
}

func newConfigListCommand(dockerCli command.Cli) *cobra.Command {
	listOpts := ListOptions{Filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List configs",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunConfigList(cmd.Context(), dockerCli, listOpts)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listOpts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&listOpts.Format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&listOpts.Filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

// RunConfigList lists Swarm configs.
func RunConfigList(ctx context.Context, dockerCLI command.Cli, options ListOptions) error {
	apiClient := dockerCLI.Client()

	configs, err := apiClient.ConfigList(ctx, types.ConfigListOptions{Filters: options.Filter.Value()})
	if err != nil {
		return err
	}

	format := options.Format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ConfigFormat) > 0 && !options.Quiet {
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
		Format: NewFormat(format, options.Quiet),
	}
	return FormatWrite(configCtx, configs)
}
