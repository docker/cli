package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts options.Remove

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespaces = args
			if err := validateStackNames(opts.Namespaces); err != nil {
				return err
			}
			return swarm.RunRemove(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.Detach, "detach", "d", true, "Do not wait for stack removal")
	return cmd
}
