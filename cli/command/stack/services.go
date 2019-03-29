package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newServicesCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	opts := options.Services{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "services [OPTIONS] STACK",
		Short: "List the services in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}
			return RunServices(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&opts.Format, "format", "", "Pretty-print services using a Go template")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunServices performs a stack services against the specified orchestrator
func RunServices(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Services) error {
	return runOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunServices(dockerCli, opts) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunServices(kli, opts) })
}
