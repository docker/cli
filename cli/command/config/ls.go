package config

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
	"vbom.ml/util/sortorder"
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
			return RunConfigList(dockerCli, listOpts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listOpts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVarP(&listOpts.Format, "format", "", "", "Pretty-print configs using a Go template")
	flags.VarP(&listOpts.Filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

// RunConfigList lists Swarm configs.
func RunConfigList(dockerCli command.Cli, options ListOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	configs, err := client.ConfigList(ctx, types.ConfigListOptions{Filters: options.Filter.Value()})
	if err != nil {
		return err
	}

	format := options.Format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ConfigFormat) > 0 && !options.Quiet {
			format = dockerCli.ConfigFile().ConfigFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(configs, func(i, j int) bool {
		return sortorder.NaturalLess(configs[i].Spec.Name, configs[j].Spec.Name)
	})

	configCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(format, options.Quiet),
	}
	return FormatWrite(configCtx, configs)
}
