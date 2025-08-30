package stack

import (
	"context"
	"io"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

// listOptions holds docker stack ls options
type listOptions struct {
	format string
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	opts := listOptions{}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

// runList performs a stack list against the specified swarm cluster
func runList(ctx context.Context, dockerCLI command.Cli, opts listOptions) error {
	stacks, err := swarm.GetStacks(ctx, dockerCLI.Client())
	if err != nil {
		return err
	}
	return format(dockerCLI.Out(), opts, stacks)
}

func format(out io.Writer, opts listOptions, stacks []formatter.Stack) error {
	fmt := formatter.Format(opts.format)
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
