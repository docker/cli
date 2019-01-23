package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newRemoveCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
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
			return RunRemove(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}
	flags := cmd.Flags()
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunRemove performs a stack remove against the specified orchestrator
func RunRemove(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Remove) error {
	return runOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunRemove(dockerCli, opts) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunRemove(kli, opts) })
}
