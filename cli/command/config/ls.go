package config

import (
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"vbom.ml/util/sortorder"
)

type byConfigName []swarm.Config

func (r byConfigName) Len() int      { return len(r) }
func (r byConfigName) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r byConfigName) Less(i, j int) bool {
	return sortorder.NaturalLess(r[i].Spec.Name, r[j].Spec.Name)
}

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newConfigListCommand(dockerCli command.Cli) *cobra.Command {
	listOpts := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List configs",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(dockerCli, listOpts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listOpts.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVarP(&listOpts.format, "format", "", "", "Pretty-print configs using a Go template")
	flags.VarP(&listOpts.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runConfigList(dockerCli command.Cli, options listOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	configs, err := client.ConfigList(ctx, types.ConfigListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ConfigFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().ConfigFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Sort(byConfigName(configs))

	configCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewConfigFormat(format, options.quiet),
	}
	return formatter.ConfigWrite(configCtx, configs)
}
