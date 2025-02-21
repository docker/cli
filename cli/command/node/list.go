package node // import "docker.com/cli/v28/cli/command/node"

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
	"github.com/docker/docker/api/types/system"
	"github.com/fvbommel/sortorder"
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
		ValidArgsFunction: completion.NoComplete,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	flags.VisitAll(func(flag *pflag.Flag) {
		// Set a default completion function if none was set. We don't look
		// up if it does already have one set, because Cobra does this for
		// us, and returns an error (which we ignore for this reason).
		_ = cmd.RegisterFlagCompletionFunc(flag.Name, completion.NoComplete)
	})
	return cmd
}

func runList(ctx context.Context, dockerCli command.Cli, options listOptions) error {
	client := dockerCli.Client()

	nodes, err := client.NodeList(
		ctx,
		types.NodeListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}

	info := system.Info{}
	if len(nodes) > 0 && !options.quiet {
		// only non-empty nodes and not quiet, should we call /info api
		info, err = client.Info(ctx)
		if err != nil {
			return err
		}
	}

	format := options.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
		if len(dockerCli.ConfigFile().NodesFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().NodesFormat
		}
	}

	nodesCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(format, options.quiet),
	}
	sort.Slice(nodes, func(i, j int) bool {
		return sortorder.NaturalLess(nodes[i].Description.Hostname, nodes[j].Description.Hostname)
	})
	return FormatWrite(nodesCtx, nodes, info)
}
