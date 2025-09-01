package node

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List nodes in the swarm",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCli, options)
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	flags.VisitAll(func(flag *pflag.Flag) {
		// Set a default completion function if none was set. We don't look
		// up if it does already have one set, because Cobra does this for
		// us, and returns an error (which we ignore for this reason).
		_ = cmd.RegisterFlagCompletionFunc(flag.Name, cobra.NoFileCompletions)
	})
	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()

	nodes, err := apiClient.NodeList(ctx, client.NodeListOptions{
		Filters: options.filter.Value(),
	})
	if err != nil {
		return err
	}

	var info system.Info
	if len(nodes) > 0 && !options.quiet {
		// only non-empty nodes and not quiet, should we call /info api
		info, err = apiClient.Info(ctx)
		if err != nil {
			return err
		}
	}

	format := options.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
		if len(dockerCLI.ConfigFile().NodesFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().NodesFormat
		}
	}

	nodesCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
	}
	sort.Slice(nodes, func(i, j int) bool {
		return sortorder.NaturalLess(nodes[i].Description.Hostname, nodes[j].Description.Hostname)
	})
	return formatWrite(nodesCtx, nodes, info)
}
