package stack

import (
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

func newListCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	opts := options.List{}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunList(cmd, dockerCli, opts, common.orchestrator)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Format, "format", "", "Pretty-print stacks using a Go template")
	return cmd
}

// RunList performs a stack list against the specified orchestrator
func RunList(cmd *cobra.Command, dockerCli command.Cli, opts options.List, orchestrator command.Orchestrator) error {
	stacks := []*formatter.Stack{}
	if orchestrator.HasSwarm() {
		ss, err := swarm.GetStacks(dockerCli)
		if err != nil {
			return err
		}
		stacks = append(stacks, ss...)
	}
	return format(dockerCli, opts, stacks)
}

func format(dockerCli command.Cli, opts options.List, stacks []*formatter.Stack) error {
	format := formatter.Format(opts.Format)
	if format == "" || format == formatter.TableFormatKey {
		format = formatter.SwarmStackTableFormat
	}
	stackCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: format,
	}
	sort.Slice(stacks, func(i, j int) bool {
		return sortorder.NaturalLess(stacks[i].Name, stacks[j].Name) ||
			!sortorder.NaturalLess(stacks[j].Name, stacks[i].Name)
	})
	return formatter.StackWrite(stackCtx, stacks)
}
