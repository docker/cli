package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
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
			if dockerCli.ClientInfo().HasKubernetes() {
				kli, err := kubernetes.WrapCli(dockerCli, cmd)
				if err != nil {
					return err
				}
				return kubernetes.RunPS(kli, opts)
			}
			return swarm.RunPS(dockerCli, opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&opts.NoTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.NoResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	flags.SetAnnotation("filter", "swarm", nil)
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&opts.Format, "format", "", "Pretty-print tasks using a Go template")

	return cmd
}
