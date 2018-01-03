package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

func newListCommand(dockerCli command.Cli) *cobra.Command {
	opts := options.List{}

	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List stacks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dockerCli.ClientInfo().HasKubernetes() {
				kli, err := kubernetes.WrapCli(dockerCli, cmd)
				if err != nil {
					return err
				}
				return kubernetes.RunList(kli, opts)
			}
			return swarm.RunList(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Format, "format", "", "Pretty-print stacks using a Go template")
	return cmd
}
