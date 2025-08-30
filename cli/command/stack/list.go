package stack

import (
	"context"
	"io"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

type listOptions = options.List

func newListCommand(dockerCli command.Cli) *cobra.Command {
	opts := listOptions{}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

// RunList performs a stack list against the specified swarm cluster
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunList(ctx context.Context, dockerCLI command.Cli, opts options.List) error {
	return runList(ctx, dockerCLI, opts)
}

// runList performs a stack list against the specified swarm cluster
func runList(ctx context.Context, dockerCLI command.Cli, opts listOptions) error {
	ss, err := swarm.GetStacks(ctx, dockerCLI.Client())
	if err != nil {
		return err
	}
	stacks := make([]*formatter.Stack, 0, len(ss))
	stacks = append(stacks, ss...)
	return format(dockerCLI.Out(), opts, stacks)
}

func format(out io.Writer, opts listOptions, stacks []*formatter.Stack) error {
	fmt := formatter.Format(opts.Format)
	if fmt == "" || fmt == formatter.TableFormatKey {
		fmt = formatter.SwarmStackTableFormat
	}
	stackCtx := formatter.Context{
		Output: out,
		Format: fmt,
	}
	sort.Slice(stacks, func(i, j int) bool {
		return sortorder.NaturalLess(stacks[i].Name, stacks[j].Name)
	})
	return formatter.StackWrite(stackCtx, stacks)
}
