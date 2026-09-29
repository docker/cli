// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package stack

import (
	"context"
	"slices"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
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
	stacks, err := getStacks(ctx, dockerCLI.Client())
	if err != nil {
		return err
	}

	format := formatter.Format(opts.format)
	if format == "" || format == formatter.TableFormatKey {
		format = stackTableFormat
	}
	stackCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: format,
	}
	slices.SortFunc(stacks, func(a, b stackSummary) int {
		return sortorder.NaturalCompare(a.Name, b.Name)
	})
	return stackWrite(stackCtx, stacks)
}
