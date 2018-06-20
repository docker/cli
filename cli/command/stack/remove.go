package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	var opts options.Remove

	cmd := &cobra.Command{
		Use:     "rm STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespaces = args
			switch {
			case common.orchestrator.HasAll():
				return errUnsupportedAllOrchestrator
			case common.orchestrator.HasKubernetes():
				kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(cmd.Flags(), common.orchestrator))
				if err != nil {
					return err
				}
				return kubernetes.RunRemove(kli, opts)
			default:
				return swarm.RunRemove(dockerCli, opts)
			}
		},
	}
	flags := cmd.Flags()
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}
