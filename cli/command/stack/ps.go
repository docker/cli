package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	flagsHelper "github.com/docker/cli/cli/flags"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

func newPsCommand(dockerCli command.Cli) *cobra.Command {
	opts := options.PS{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}
			return swarm.RunPS(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&opts.NoTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.NoResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&opts.Format, "format", "", flagsHelper.FormatHelp)
	return cmd
}
